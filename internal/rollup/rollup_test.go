package rollup

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ramhandean/diorama-ops/internal/db"
	"github.com/ramhandean/diorama-ops/internal/tenants"
)

func TestCleanPathAndReferrer(t *testing.T) {
	if p := CleanPath("/blog/post-1?utm_source=twitter#comments"); p != "/blog/post-1" {
		t.Fatalf("unexpected cleaned path: %s", p)
	}
	if p := CleanPath("docs/page"); p != "/docs/page" {
		t.Fatalf("expected leading slash: %s", p)
	}

	if r := CleanReferrer("https://news.ycombinator.com/item?id=12345"); r != "news.ycombinator.com" {
		t.Fatalf("unexpected referrer: %s", r)
	}
	if r := CleanReferrer("http://localhost:8080/dashboard"); r != "" {
		t.Fatalf("expected localhost referrer to be stripped, got %s", r)
	}
}

func TestRollupAggregation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dioramaops-rollup-test-*")
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
	// Insert dummy tenant
	tStore := tenants.NewStore(dbConn)
	_ = tStore.Create(ctx, &tenants.Tenant{
		ID:        "app",
		Name:      "App",
		URL:       "https://app.io",
		CreatedAt: time.Now(),
	})

	batch := db.NewBatchWriter(dbConn, 10*time.Millisecond, 100)
	defer batch.Close()

	engine := NewEngine(dbConn, batch, time.UTC, 30)
	defer engine.Close()

	// Record pageview and heartbeat
	engine.RecordPageview("app", "/dashboard?tab=1", "https://google.com/search?q=app", true)
	engine.RecordHeartbeat("app")

	// Flush batch to SQLite
	if err := batch.Flush(); err != nil {
		t.Fatalf("flush batch: %v", err)
	}

	// Verify hits_hourly
	var pageviews, sessions, dwellSum int
	err = dbConn.QueryRowContext(ctx, "SELECT pageviews, sessions, dwell_sum FROM hits_hourly WHERE tenant_id = 'app'").Scan(&pageviews, &sessions, &dwellSum)
	if err != nil {
		t.Fatalf("query hits_hourly: %v", err)
	}

	if pageviews != 1 || sessions != 1 || dwellSum != 15 {
		t.Fatalf("expected pv=1, sess=1, dwell=15; got pv=%d, sess=%d, dwell=%d", pageviews, sessions, dwellSum)
	}

	// Verify paths_daily
	var pathHits int
	err = dbConn.QueryRowContext(ctx, "SELECT hits FROM paths_daily WHERE tenant_id = 'app' AND path = '/dashboard'").Scan(&pathHits)
	if err != nil {
		t.Fatalf("query paths_daily: %v", err)
	}
	if pathHits != 1 {
		t.Fatalf("expected path hits=1, got %d", pathHits)
	}

	// Verify referrers_daily
	var refHits int
	err = dbConn.QueryRowContext(ctx, "SELECT hits FROM referrers_daily WHERE tenant_id = 'app' AND domain = 'google.com'").Scan(&refHits)
	if err != nil {
		t.Fatalf("query referrers_daily: %v", err)
	}
	if refHits != 1 {
		t.Fatalf("expected ref hits=1, got %d", refHits)
	}
}
