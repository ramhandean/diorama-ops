package presence

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

type EventType string

const (
	EventEnter EventType = "enter"
	EventLeave EventType = "leave"
)

type PresenceEvent struct {
	Type     EventType
	TenantID string
	VID      string
}

type Session struct {
	SID       string
	VID       string
	TenantID  string
	Path      string
	Ref       string
	LastSeen  time.Time
	Heartbeat int
}

type Tracker struct {
	ttl       time.Duration
	tz        *time.Location
	mu        sync.RWMutex
	sessions  map[string]*Session          // key: sid
	tenantVID map[string]map[string]int    // tenantID -> vid -> count of active tabs (sid)

	saltMu    sync.RWMutex
	dailySalt []byte
	currentDay string

	listenersMu sync.RWMutex
	listeners   []chan PresenceEvent

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewTracker(ttl time.Duration, tz *time.Location) *Tracker {
	if ttl <= 0 {
		ttl = 45 * time.Second
	}
	if tz == nil {
		tz = time.UTC
	}

	t := &Tracker{
		ttl:        ttl,
		tz:         tz,
		sessions:   make(map[string]*Session),
		tenantVID:  make(map[string]map[string]int),
		dailySalt:  generateSalt(),
		currentDay: time.Now().In(tz).Format("2006-01-02"),
		stopCh:     make(chan struct{}),
	}

	t.wg.Add(1)
	go t.sweepLoop()
	return t
}

func generateSalt() []byte {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return b
}

func (t *Tracker) HashVID(ip, ua, siteKey string) string {
	t.saltMu.Lock()
	today := time.Now().In(t.tz).Format("2006-01-02")
	if today != t.currentDay {
		t.dailySalt = generateSalt()
		t.currentDay = today
	}
	salt := t.dailySalt
	t.saltMu.Unlock()

	mac := hmac.New(sha256.New, salt)
	mac.Write([]byte(ip))
	mac.Write([]byte("|"))
	mac.Write([]byte(ua))
	mac.Write([]byte("|"))
	mac.Write([]byte(siteKey))
	return "v_" + hex.EncodeToString(mac.Sum(nil))[:16]
}

func (t *Tracker) Subscribe() <-chan PresenceEvent {
	t.listenersMu.Lock()
	defer t.listenersMu.Unlock()
	ch := make(chan PresenceEvent, 256)
	t.listeners = append(t.listeners, ch)
	return ch
}

func (t *Tracker) Unsubscribe(ch <-chan PresenceEvent) {
	t.listenersMu.Lock()
	defer t.listenersMu.Unlock()
	for i, l := range t.listeners {
		if l == ch {
			t.listeners = append(t.listeners[:i], t.listeners[i+1:]...)
			close(l)
			break
		}
	}
}

func (t *Tracker) broadcast(event PresenceEvent) {
	t.listenersMu.RLock()
	defer t.listenersMu.RUnlock()
	for _, ch := range t.listeners {
		select {
		case ch <- event:
		default:
		}
	}
}

func (t *Tracker) Touch(sid, vid, tenantID, path, ref string) (isNewVID bool, isNewSID bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	sess, exists := t.sessions[sid]
	if !exists {
		isNewSID = true
		sess = &Session{
			SID:      sid,
			VID:      vid,
			TenantID: tenantID,
			Path:     path,
			Ref:      ref,
			LastSeen: now,
		}
		t.sessions[sid] = sess

		vids, ok := t.tenantVID[tenantID]
		if !ok {
			vids = make(map[string]int)
			t.tenantVID[tenantID] = vids
		}
		vids[vid]++
		if vids[vid] == 1 {
			isNewVID = true
			go t.broadcast(PresenceEvent{
				Type:     EventEnter,
				TenantID: tenantID,
				VID:      vid,
			})
		}
	} else {
		sess.LastSeen = now
		sess.Heartbeat++
		if path != "" {
			sess.Path = path
		}
	}

	return isNewVID, isNewSID
}

func (t *Tracker) TouchHB(sid, path string) *Session {
	t.mu.Lock()
	defer t.mu.Unlock()

	sess, exists := t.sessions[sid]
	if !exists {
		return nil
	}
	sess.LastSeen = time.Now()
	sess.Heartbeat++
	if path != "" {
		sess.Path = path
	}
	return sess
}

func (t *Tracker) TouchNav(sid, path string) *Session {
	t.mu.Lock()
	defer t.mu.Unlock()

	sess, exists := t.sessions[sid]
	if !exists {
		return nil
	}
	sess.LastSeen = time.Now()
	if path != "" {
		sess.Path = path
	}
	return sess
}

func (t *Tracker) Remove(sid string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	sess, exists := t.sessions[sid]
	if !exists {
		return
	}

	delete(t.sessions, sid)
	t.decrementVID(sess.TenantID, sess.VID)
}

func (t *Tracker) decrementVID(tenantID, vid string) {
	vids, ok := t.tenantVID[tenantID]
	if !ok {
		return
	}
	vids[vid]--
	if vids[vid] <= 0 {
		delete(vids, vid)
		if len(vids) == 0 {
			delete(t.tenantVID, tenantID)
		}
		go t.broadcast(PresenceEvent{
			Type:     EventLeave,
			TenantID: tenantID,
			VID:      vid,
		})
	}
}

func (t *Tracker) GetActiveCounts() map[string]int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	counts := make(map[string]int, len(t.tenantVID))
	for tid, vids := range t.tenantVID {
		counts[tid] = len(vids)
	}
	return counts
}

func (t *Tracker) GetTenantActiveCount(tenantID string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.tenantVID[tenantID])
}

func (t *Tracker) TotalActiveVisitors() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	total := 0
	for _, vids := range t.tenantVID {
		total += len(vids)
	}
	return total
}

func (t *Tracker) sweepLoop() {
	defer t.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.sweep()
		case <-t.stopCh:
			return
		}
	}
}

func (t *Tracker) sweep() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for sid, sess := range t.sessions {
		if now.Sub(sess.LastSeen) > t.ttl {
			delete(t.sessions, sid)
			t.decrementVID(sess.TenantID, sess.VID)
		}
	}
}

func (t *Tracker) Close() {
	close(t.stopCh)
	t.wg.Wait()
}
