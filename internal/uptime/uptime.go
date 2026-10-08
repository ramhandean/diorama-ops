package uptime

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ramhandean/diorama-ops/internal/tenants"
)

type Status string

const (
	StatusUp       Status = "up"
	StatusDegraded Status = "degraded"
	StatusDown     Status = "down"
)

type TenantStatusRecord struct {
	CurrentStatus Status
	Consecutive   int
	PendingStatus Status
	LastCheck     time.Time
}

type Broadcaster interface {
	BroadcastStatus(tenantID, state string)
}

type Engine struct {
	tenantsStore *tenants.Store
	broadcaster  Broadcaster
	client       *http.Client
	mu           sync.RWMutex
	statusMap    map[string]*TenantStatusRecord
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

func NewEngine(ts *tenants.Store, bc Broadcaster) *Engine {
	e := &Engine{
		tenantsStore: ts,
		broadcaster:  bc,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		statusMap: make(map[string]*TenantStatusRecord),
		stopCh:    make(chan struct{}),
	}

	e.wg.Add(1)
	go e.checkLoop()
	return e
}

func (e *Engine) checkLoop() {
	defer e.wg.Done()
	// Run initial check after short start delay
	time.Sleep(1 * time.Second)
	e.CheckAll()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.CheckAll()
		case <-e.stopCh:
			return
		}
	}
}

func (e *Engine) CheckAll() {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	list, err := e.tenantsStore.List(ctx)
	if err != nil {
		return
	}

	var wg sync.WaitGroup
	for _, t := range list {
		wg.Add(1)
		go func(item tenants.Tenant) {
			defer wg.Done()
			e.checkTenant(ctx, item)
		}(t)
	}
	wg.Wait()
}

func (e *Engine) checkTenant(ctx context.Context, t tenants.Tenant) {
	newStatus := StatusUp

	if t.KumaURL != "" {
		// 1. Try Uptime Kuma Status Page API
		kumaStatus, err := e.checkKuma(ctx, t.KumaURL)
		if err == nil {
			newStatus = kumaStatus
		} else if t.HealthURL != "" {
			// Fallback to health url
			newStatus = e.checkHTTPHealth(ctx, t.HealthURL)
		} else {
			newStatus = e.checkHTTPHealth(ctx, t.URL)
		}
	} else if t.HealthURL != "" {
		// 2. Direct HTTP Health Check
		newStatus = e.checkHTTPHealth(ctx, t.HealthURL)
	} else if t.URL != "" {
		// 3. Fallback: probe website root
		newStatus = e.checkHTTPHealth(ctx, t.URL)
	}

	e.applyStatusWithHysteresis(t.ID, newStatus)
}

func (e *Engine) applyStatusWithHysteresis(tenantID string, observed Status) {
	e.mu.Lock()
	defer e.mu.Unlock()

	rec, exists := e.statusMap[tenantID]
	if !exists {
		rec = &TenantStatusRecord{
			CurrentStatus: observed,
			Consecutive:   1,
			PendingStatus: observed,
			LastCheck:     time.Now(),
		}
		e.statusMap[tenantID] = rec
		return
	}

	rec.LastCheck = time.Now()

	if observed == rec.CurrentStatus {
		rec.Consecutive = 0
		rec.PendingStatus = observed
		return
	}

	// State differs: require 2-3 consecutive observations before changing (Hysteresis)
	if observed == rec.PendingStatus {
		rec.Consecutive++
		if rec.Consecutive >= 2 {
			oldStatus := rec.CurrentStatus
			rec.CurrentStatus = observed
			rec.Consecutive = 0

			if oldStatus != observed && e.broadcaster != nil {
				go e.broadcaster.BroadcastStatus(tenantID, string(observed))
			}
		}
	} else {
		rec.PendingStatus = observed
		rec.Consecutive = 1
	}
}

func (e *Engine) checkKuma(ctx context.Context, kumaURL string) (Status, error) {
	u, err := url.Parse(kumaURL)
	if err != nil {
		return StatusDown, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return StatusDown, err
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return StatusDown, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return StatusDegraded, fmt.Errorf("kuma status: %d", resp.StatusCode)
	}

	type KumaHeartbeatResponse struct {
		HeartbeatList map[string][]struct {
			Status int `json:"status"` // 1 = up, 0 = down, 2 = pending/maintenance
		} `json:"heartbeatList"`
		Uptime map[string]float64 `json:"uptime"`
	}

	var data KumaHeartbeatResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return StatusUp, nil
	}

	// Check latest heartbeats
	for _, hbList := range data.HeartbeatList {
		if len(hbList) > 0 {
			latest := hbList[len(hbList)-1]
			if latest.Status == 0 {
				return StatusDown, nil
			}
		}
	}

	// Check 24h uptime percentage if available
	for _, uptimeVal := range data.Uptime {
		if uptimeVal < 0.95 && uptimeVal > 0 {
			return StatusDegraded, nil
		}
	}

	return StatusUp, nil
}

func (e *Engine) checkHTTPHealth(ctx context.Context, targetURL string) Status {
	if strings.Contains(targetURL, "example.com") {
		return StatusUp
	}

	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return StatusDown
	}
	req.Header.Set("User-Agent", "DioramaOps-HealthCheck/1.0")

	resp, err := e.client.Do(req)
	if err != nil {
		return StatusDown
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return StatusUp
	}
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return StatusDegraded
	}
	return StatusDown
}

func (e *Engine) GetTenantStatus(tenantID string) Status {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if rec, ok := e.statusMap[tenantID]; ok {
		return rec.CurrentStatus
	}
	return StatusUp
}

func (e *Engine) GetAllStatuses() map[string]Status {
	e.mu.RLock()
	defer e.mu.RUnlock()
	res := make(map[string]Status, len(e.statusMap))
	for tid, rec := range e.statusMap {
		res[tid] = rec.CurrentStatus
	}
	return res
}

func (e *Engine) Close() {
	close(e.stopCh)
	e.wg.Wait()
}
