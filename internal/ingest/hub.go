package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/ramhandean/diorama-ops/internal/config"
	"github.com/ramhandean/diorama-ops/internal/presence"
	"github.com/ramhandean/diorama-ops/internal/rollup"
	"github.com/ramhandean/diorama-ops/internal/tenants"
)

type ClientMessage struct {
	Type string `json:"t"`
	Key  string `json:"key,omitempty"`
	SID  string `json:"sid"`
	Path string `json:"path,omitempty"`
	Ref  string `json:"ref,omitempty"`
}

type Hub struct {
	cfg        *config.Config
	tenants    *tenants.Store
	presence   *presence.Tracker
	rollup     *rollup.Engine
	ipLimiter  *Limiter
	keyLimiter *Limiter

	dashboardsMu sync.RWMutex
	dashboards   map[*dashboardClient]bool

	sseMu      sync.RWMutex
	sseClients map[chan []byte]bool

	eventQueueMu sync.Mutex
	eventQueue   []presence.PresenceEvent

	stopCh chan struct{}
	wg     sync.WaitGroup
}

type dashboardClient struct {
	conn *websocket.Conn
	send chan []byte
}

func NewHub(cfg *config.Config, ts *tenants.Store, pt *presence.Tracker, re *rollup.Engine) *Hub {
	h := &Hub{
		cfg:        cfg,
		tenants:    ts,
		presence:   pt,
		rollup:     re,
		ipLimiter:  NewLimiter(30.0, 60.0), // 30 req/s, burst 60 per IP
		keyLimiter: NewLimiter(50.0, 100.0), // 50 req/s, burst 100 per site key
		dashboards: make(map[*dashboardClient]bool),
		sseClients: make(map[chan []byte]bool),
		stopCh:     make(chan struct{}),
	}

	h.wg.Add(2)
	go h.presenceListenerLoop()
	go h.broadcastThrottleLoop()
	return h
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	// Check if this connection is a dashboard subscriber or a site client
	isDashboard := r.URL.Query().Get("role") == "dashboard"

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // we validate origin manually for tenants
	})
	if err != nil {
		return
	}

	if isDashboard {
		h.handleDashboard(conn, r)
		return
	}

	h.handleClient(conn, r)
}

func (h *Hub) handleDashboard(conn *websocket.Conn, r *http.Request) {
	defer conn.Close(websocket.StatusNormalClosure, "")

	client := &dashboardClient{
		conn: conn,
		send: make(chan []byte, 128),
	}

	// Send initial snapshot
	snapPayload := h.buildSnapshot(r.Context())
	if err := conn.Write(r.Context(), websocket.MessageText, snapPayload); err != nil {
		return
	}

	h.dashboardsMu.Lock()
	h.dashboards[client] = true
	h.dashboardsMu.Unlock()

	defer func() {
		h.dashboardsMu.Lock()
		delete(h.dashboards, client)
		close(client.send)
		h.dashboardsMu.Unlock()
	}()

	// Reader pump to catch disconnects
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go func() {
		for {
			_, _, err := conn.Read(ctx)
			if err != nil {
				cancel()
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-client.send:
			if !ok {
				return
			}
			if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
				return
			}
		}
	}
}

func (h *Hub) handleClient(conn *websocket.Conn, r *http.Request) {
	defer conn.Close(websocket.StatusNormalClosure, "")
	conn.SetReadLimit(1024) // 1 KB max payload limit

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	ip := h.extractIP(r)
	ua := r.UserAgent()

	// DNT & bot checks
	if r.Header.Get("DNT") == "1" || isBot(ua) {
		return
	}

	var activeSID string
	defer func() {
		if activeSID != "" {
			h.presence.Remove(activeSID)
		}
	}()

	for {
		_, reader, err := conn.Reader(ctx)
		if err != nil {
			return
		}

		payload, err := io.ReadAll(io.LimitReader(reader, 1024))
		if err != nil {
			return
		}

		var msg ClientMessage
		if err := json.Unmarshal(payload, &msg); err != nil {
			continue
		}

		sid, err := h.processMessage(ctx, &msg, ip, ua, r.Header.Get("Origin"))
		if err != nil {
			continue
		}
		if sid != "" {
			activeSID = sid
		}
	}
}

func (h *Hub) ServeBeacon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Always respond 204 No Content for sendBeacon
	w.WriteHeader(http.StatusNoContent)

	if r.Header.Get("DNT") == "1" {
		return
	}

	ua := r.UserAgent()
	if isBot(ua) {
		return
	}

	ip := h.extractIP(r)
	body, err := io.ReadAll(io.LimitReader(r.Body, 1024))
	if err != nil {
		return
	}

	var msg ClientMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		return
	}

	_, _ = h.processMessage(r.Context(), &msg, ip, ua, r.Header.Get("Origin"))
}

func (h *Hub) ServeSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan []byte, 64)
	h.sseMu.Lock()
	h.sseClients[ch] = true
	h.sseMu.Unlock()

	defer func() {
		h.sseMu.Lock()
		delete(h.sseClients, ch)
		close(ch)
		h.sseMu.Unlock()
	}()

	// Send initial snapshot
	snap := h.buildSnapshot(r.Context())
	fmt.Fprintf(w, "data: %s\n\n", snap)
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

func (h *Hub) processMessage(ctx context.Context, msg *ClientMessage, ip, ua, origin string) (string, error) {
	if !h.ipLimiter.Allow(ip) {
		return "", errors.New("ip rate limited")
	}

	switch msg.Type {
	case "hello":
		if msg.Key == "" || msg.SID == "" {
			return "", errors.New("key and sid required")
		}
		if !h.keyLimiter.Allow(msg.Key) {
			return "", errors.New("key rate limited")
		}

		tenant, err := h.tenants.GetBySiteKey(ctx, msg.Key)
		if err != nil {
			return "", err
		}

		if origin != "" && !tenant.ValidateOrigin(origin) {
			return "", errors.New("origin rejected")
		}

		vid := h.presence.HashVID(ip, ua, tenant.SiteKey)
		_, isNewSID := h.presence.Touch(msg.SID, vid, tenant.ID, msg.Path, msg.Ref)
		h.rollup.RecordPageview(tenant.ID, msg.Path, msg.Ref, isNewSID)
		return msg.SID, nil

	case "hb":
		if msg.SID == "" {
			return "", errors.New("sid required")
		}
		sess := h.presence.TouchHB(msg.SID, msg.Path)
		if sess != nil {
			h.rollup.RecordHeartbeat(sess.TenantID)
			return msg.SID, nil
		}
		return "", errors.New("unknown session")

	case "nav":
		if msg.SID == "" {
			return "", errors.New("sid required")
		}
		sess := h.presence.TouchNav(msg.SID, msg.Path)
		if sess != nil {
			h.rollup.RecordPageview(sess.TenantID, msg.Path, "", false)
			return msg.SID, nil
		}
		return "", errors.New("unknown session")

	case "bye":
		if msg.SID != "" {
			h.presence.Remove(msg.SID)
		}
		return "", nil

	default:
		return "", errors.New("unknown message type")
	}
}

func (h *Hub) presenceListenerLoop() {
	defer h.wg.Done()
	ch := h.presence.Subscribe()
	defer h.presence.Unsubscribe(ch)

	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return
			}
			h.eventQueueMu.Lock()
			h.eventQueue = append(h.eventQueue, ev)
			h.eventQueueMu.Unlock()
		case <-h.stopCh:
			return
		}
	}
}

func (h *Hub) broadcastThrottleLoop() {
	defer h.wg.Done()
	ticker := time.NewTicker(150 * time.Millisecond) // ~6.6 Hz throttling
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.flushBroadcast()
		case <-h.stopCh:
			return
		}
	}
}

func (h *Hub) flushBroadcast() {
	h.eventQueueMu.Lock()
	if len(h.eventQueue) == 0 {
		h.eventQueueMu.Unlock()
		return
	}
	events := h.eventQueue
	h.eventQueue = nil
	h.eventQueueMu.Unlock()

	for _, ev := range events {
		payload, err := json.Marshal(map[string]interface{}{
			"t":      string(ev.Type),
			"tenant": ev.TenantID,
			"vid":    ev.VID,
		})
		if err != nil {
			continue
		}
		h.broadcastToDashboards(payload)
	}
}

func (h *Hub) BroadcastStatus(tenantID, state string) {
	payload, err := json.Marshal(map[string]interface{}{
		"t":      "status",
		"tenant": tenantID,
		"state":  state,
	})
	if err != nil {
		return
	}
	h.broadcastToDashboards(payload)
}

func (h *Hub) broadcastToDashboards(msg []byte) {
	h.dashboardsMu.RLock()
	for client := range h.dashboards {
		select {
		case client.send <- msg:
		default:
		}
	}
	h.dashboardsMu.RUnlock()

	h.sseMu.RLock()
	for ch := range h.sseClients {
		select {
		case ch <- msg:
		default:
		}
	}
	h.sseMu.RUnlock()
}

func (h *Hub) buildSnapshot(ctx context.Context) []byte {
	list, err := h.tenants.List(ctx)
	if err != nil {
		list = []tenants.Tenant{}
	}

	type tenantSnap struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Active int    `json:"active"`
		Status string `json:"status"`
	}

	counts := h.presence.GetActiveCounts()
	snaps := make([]tenantSnap, 0, len(list))
	for _, t := range list {
		snaps = append(snaps, tenantSnap{
			ID:     t.ID,
			Name:   t.Name,
			Active: counts[t.ID],
			Status: "up", // Uptime engine will integrate here in Phase 3
		})
	}

	data, _ := json.Marshal(map[string]interface{}{
		"t":       "snap",
		"tenants": snaps,
	})
	return data
}

func (h *Hub) extractIP(r *http.Request) string {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteIP = r.RemoteAddr
	}

	if h.cfg.IsTrustedProxy(remoteIP) {
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			parts := strings.Split(xff, ",")
			clientIP := strings.TrimSpace(parts[0])
			if net.ParseIP(clientIP) != nil {
				return clientIP
			}
		}
		xri := r.Header.Get("X-Real-IP")
		if xri != "" && net.ParseIP(strings.TrimSpace(xri)) != nil {
			return strings.TrimSpace(xri)
		}
	}
	return remoteIP
}

func isBot(ua string) bool {
	lower := strings.ToLower(ua)
	return strings.Contains(lower, "bot") ||
		strings.Contains(lower, "crawl") ||
		strings.Contains(lower, "spider") ||
		strings.Contains(lower, "slurp") ||
		strings.Contains(lower, "headless") ||
		strings.Contains(lower, "curl") ||
		strings.Contains(lower, "wget")
}

func (h *Hub) Close() {
	close(h.stopCh)
	h.ipLimiter.Close()
	h.keyLimiter.Close()
	h.wg.Wait()
}
