package ingest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/ramhandean/diorama-ops/internal/config"
	"github.com/ramhandean/diorama-ops/internal/db"
	"github.com/ramhandean/diorama-ops/internal/presence"
	"github.com/ramhandean/diorama-ops/internal/rollup"
	"github.com/ramhandean/diorama-ops/internal/tenants"
)

func TestEndToEndTelemetryFlow(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dioramaops-ingest-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbConn, err := db.Open(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer dbConn.Close()

	ctx := context.Background()

	// Seed tenant
	tenantStore := tenants.NewStore(dbConn)
	siteKey := "pk_test_ingest_key_123"
	tenantID := "demo-site"
	err = tenantStore.Create(ctx, &tenants.Tenant{
		ID:        tenantID,
		Name:      "Demo Site",
		URL:       "https://demo.local",
		SiteKey:   siteKey,
		Origins:   []string{"http://demo.local:3000"},
		CreatedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("seed tenant: %v", err)
	}

	batchWriter := db.NewBatchWriter(dbConn, 10*time.Millisecond, 100)
	defer batchWriter.Close()

	presenceTracker := presence.NewTracker(1*time.Second, time.UTC)
	defer presenceTracker.Close()

	rollupEngine := rollup.NewEngine(dbConn, batchWriter, time.UTC, 30)
	defer rollupEngine.Close()

	cfg := &config.Config{
		TZ: time.UTC,
	}

	hub := NewHub(cfg, tenantStore, presenceTracker, rollupEngine)
	defer hub.Close()

	// Start test HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.ServeWS)
	mux.HandleFunc("POST /api/v1/b", hub.ServeBeacon)
	server := httptest.NewServer(mux)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// 1. Connect WS with valid origin
	wsHeaders := http.Header{}
	wsHeaders.Set("Origin", "http://demo.local:3000")
	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: wsHeaders,
	})
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// 2. Send hello
	sid := "sid_987654321"
	helloMsg := `{"t":"hello","key":"` + siteKey + `","sid":"` + sid + `","path":"/pricing?ref=twitter","ref":"twitter.com"}`
	if err := conn.Write(ctx, websocket.MessageText, []byte(helloMsg)); err != nil {
		t.Fatalf("send hello: %v", err)
	}

	// Wait briefly for presence touch
	time.Sleep(50 * time.Millisecond)
	if count := presenceTracker.GetTenantActiveCount(tenantID); count != 1 {
		t.Fatalf("expected 1 active visitor, got %d", count)
	}

	// 3. Send heartbeat
	hbMsg := `{"t":"hb","sid":"` + sid + `","path":"/pricing"}`
	if err := conn.Write(ctx, websocket.MessageText, []byte(hbMsg)); err != nil {
		t.Fatalf("send hb: %v", err)
	}

	// 4. Send bye
	byeMsg := `{"t":"bye","sid":"` + sid + `"}`
	if err := conn.Write(ctx, websocket.MessageText, []byte(byeMsg)); err != nil {
		t.Fatalf("send bye: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if count := presenceTracker.GetTenantActiveCount(tenantID); count != 0 {
		t.Fatalf("expected 0 active visitors after bye, got %d", count)
	}

	// 5. Flush batch to SQLite and verify rollup
	if err := batchWriter.Flush(); err != nil {
		t.Fatalf("flush batch: %v", err)
	}

	var pageviews, sessions, dwellSum int
	err = dbConn.QueryRowContext(ctx, "SELECT pageviews, sessions, dwell_sum FROM hits_hourly WHERE tenant_id = ?", tenantID).Scan(&pageviews, &sessions, &dwellSum)
	if err != nil {
		t.Fatalf("query hits_hourly: %v", err)
	}

	if pageviews != 1 || sessions != 1 || dwellSum != 15 {
		t.Fatalf("expected pv=1, sess=1, dwell=15; got pv=%d, sess=%d, dwell=%d", pageviews, sessions, dwellSum)
	}

	// Verify paths_daily recorded cleaned path /pricing
	var pathHits int
	err = dbConn.QueryRowContext(ctx, "SELECT hits FROM paths_daily WHERE tenant_id = ? AND path = '/pricing'", tenantID).Scan(&pathHits)
	if err != nil {
		t.Fatalf("query paths_daily: %v", err)
	}
	if pathHits != 1 {
		t.Fatalf("expected path hits=1, got %d", pathHits)
	}
}

func TestBeaconFallback(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dioramaops-beacon-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbConn, err := db.Open(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer dbConn.Close()

	ctx := context.Background()
	tenantStore := tenants.NewStore(dbConn)
	siteKey := "pk_beacon_key_456"
	tenantID := "beacon-site"
	_ = tenantStore.Create(ctx, &tenants.Tenant{
		ID:        tenantID,
		Name:      "Beacon Site",
		URL:       "https://beacon.local",
		SiteKey:   siteKey,
		Origins:   []string{"http://beacon.local"},
		CreatedAt: time.Now(),
	})

	batchWriter := db.NewBatchWriter(dbConn, 10*time.Millisecond, 100)
	defer batchWriter.Close()

	presenceTracker := presence.NewTracker(1*time.Second, time.UTC)
	defer presenceTracker.Close()

	rollupEngine := rollup.NewEngine(dbConn, batchWriter, time.UTC, 30)
	defer rollupEngine.Close()

	hub := NewHub(&config.Config{TZ: time.UTC}, tenantStore, presenceTracker, rollupEngine)
	defer hub.Close()

	// POST /api/v1/b
	beaconReqBody := `{"t":"hello","key":"` + siteKey + `","sid":"s_beacon_01","path":"/","ref":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/b", strings.NewReader(beaconReqBody))
	req.Header.Set("Origin", "http://beacon.local")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	hub.ServeBeacon(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected CORS Access-Control-Allow-Origin: *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}

	// OPTIONS preflight
	optReq := httptest.NewRequest(http.MethodOptions, "/api/v1/b", nil)
	optW := httptest.NewRecorder()
	hub.ServeBeacon(optW, optReq)
	if optW.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS preflight, got %d", optW.Code)
	}

	if count := presenceTracker.GetTenantActiveCount(tenantID); count != 1 {
		t.Fatalf("expected 1 active visitor via beacon, got %d", count)
	}
}
