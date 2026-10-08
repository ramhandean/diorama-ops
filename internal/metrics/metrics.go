package metrics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/ramhandean/diorama-ops/internal/config"
	"github.com/ramhandean/diorama-ops/internal/db"
	"github.com/ramhandean/diorama-ops/internal/presence"
	"github.com/ramhandean/diorama-ops/internal/tenants"
	"github.com/ramhandean/diorama-ops/internal/uptime"
)

type Registry struct {
	cfg          *config.Config
	db           *db.DB
	tenantsStore *tenants.Store
	presence     *presence.Tracker
	uptime       *uptime.Engine

	droppedDroppedMu sync.RWMutex
	droppedCounters  map[string]*int64

	wsConnections int64
}

func NewRegistry(
	cfg *config.Config,
	dbConn *db.DB,
	ts *tenants.Store,
	pt *presence.Tracker,
	ue *uptime.Engine,
) *Registry {
	return &Registry{
		cfg:             cfg,
		db:              dbConn,
		tenantsStore:    ts,
		presence:        pt,
		uptime:          ue,
		droppedCounters: make(map[string]*int64),
	}
}

func (r *Registry) IncDropped(reason string) {
	r.droppedDroppedMu.Lock()
	counter, ok := r.droppedCounters[reason]
	if !ok {
		var n int64
		counter = &n
		r.droppedCounters[reason] = counter
	}
	r.droppedDroppedMu.Unlock()
	atomic.AddInt64(counter, 1)
}

func (r *Registry) SetWSConnections(count int64) {
	atomic.StoreInt64(&r.wsConnections, count)
}

func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Authentication check if METRICS_TOKEN is set
	if r.cfg.MetricsToken != "" {
		auth := req.Header.Get("Authorization")
		token := ""
		if len(auth) > 7 && auth[:7] == "Bearer " {
			token = auth[7:]
		}
		if token != r.cfg.MetricsToken {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	r.writeMetrics(w, req.Context())
}

func (r *Registry) writeMetrics(w io.Writer, ctx context.Context) {
	list, err := r.tenantsStore.List(ctx)
	if err != nil {
		list = []tenants.Tenant{}
	}

	// 1. Header & HELP comments
	fmt.Fprintln(w, "# HELP dioramaops_ws_connections Current number of active dashboard/client WebSocket connections")
	fmt.Fprintln(w, "# TYPE dioramaops_ws_connections gauge")
	fmt.Fprintf(w, "dioramaops_ws_connections %d\n\n", atomic.LoadInt64(&r.wsConnections))

	fmt.Fprintln(w, "# HELP dioramaops_active_visitors Current unique active visitors per tenant")
	fmt.Fprintln(w, "# TYPE dioramaops_active_visitors gauge")
	for _, t := range list {
		active := r.presence.GetTenantActiveCount(t.ID)
		fmt.Fprintf(w, "dioramaops_active_visitors{tenant=%q} %d\n", t.ID, active)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "# HELP dioramaops_tenant_up Operational status of tenant booth (1 = up, 0 = degraded/down)")
	fmt.Fprintln(w, "# TYPE dioramaops_tenant_up gauge")
	for _, t := range list {
		st := r.uptime.GetTenantStatus(t.ID)
		val := 0
		if st == uptime.StatusUp {
			val = 1
		}
		fmt.Fprintf(w, "dioramaops_tenant_up{tenant=%q} %d\n", t.ID, val)
	}
	fmt.Fprintln(w)

	// Historical aggregated hits, sessions, avg dwell from database
	fmt.Fprintln(w, "# HELP dioramaops_hits_total Total pageviews recorded for tenant")
	fmt.Fprintln(w, "# TYPE dioramaops_hits_total counter")
	fmt.Fprintln(w, "# HELP dioramaops_sessions_total Total sessions recorded for tenant")
	fmt.Fprintln(w, "# TYPE dioramaops_sessions_total counter")
	fmt.Fprintln(w, "# HELP dioramaops_avg_dwell_seconds Average dwell time in seconds for tenant")
	fmt.Fprintln(w, "# TYPE dioramaops_avg_dwell_seconds gauge")

	for _, t := range list {
		var hits, sessions, dwellSum int64
		row := r.db.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(pageviews), 0), COALESCE(SUM(sessions), 0), COALESCE(SUM(dwell_sum), 0)
			FROM hits_hourly
			WHERE tenant_id = ?
		`, t.ID)
		_ = row.Scan(&hits, &sessions, &dwellSum)

		fmt.Fprintf(w, "dioramaops_hits_total{tenant=%q} %d\n", t.ID, hits)
		fmt.Fprintf(w, "dioramaops_sessions_total{tenant=%q} %d\n", t.ID, sessions)

		avgDwell := 0.0
		if sessions > 0 {
			avgDwell = float64(dwellSum) / float64(sessions)
		}
		fmt.Fprintf(w, "dioramaops_avg_dwell_seconds{tenant=%q} %.2f\n", t.ID, avgDwell)
	}
	fmt.Fprintln(w)

	// Ingest dropped reasons
	fmt.Fprintln(w, "# HELP dioramaops_ingest_dropped_total Total telemetry requests dropped by reason")
	fmt.Fprintln(w, "# TYPE dioramaops_ingest_dropped_total counter")
	r.droppedDroppedMu.RLock()
	for reason, ptr := range r.droppedCounters {
		fmt.Fprintf(w, "dioramaops_ingest_dropped_total{reason=%q} %d\n", reason, atomic.LoadInt64(ptr))
	}
	r.droppedDroppedMu.RUnlock()
}
