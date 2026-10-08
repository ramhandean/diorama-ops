package rollup

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ramhandean/diorama-ops/internal/db"
)

type Engine struct {
	db          *db.DB
	batch       *db.BatchWriter
	tz          *time.Location
	retention   int // in days
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

func NewEngine(dbConn *db.DB, batchWriter *db.BatchWriter, tz *time.Location, retentionDays int) *Engine {
	if tz == nil {
		tz = time.UTC
	}
	if retentionDays <= 0 {
		retentionDays = 395
	}

	e := &Engine{
		db:        dbConn,
		batch:     batchWriter,
		tz:        tz,
		retention: retentionDays,
		stopCh:    make(chan struct{}),
	}

	e.wg.Add(1)
	go e.pruneLoop()
	return e
}

func (e *Engine) RecordPageview(tenantID, rawPath, rawRef string, isNewSession bool) {
	now := time.Now().In(e.tz)
	bucketTS := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, e.tz).Unix()
	dayStr := now.Format("2006-01-02")

	sessionInc := 0
	if isNewSession {
		sessionInc = 1
	}

	// 1. Record hourly hit (1 pageview)
	e.batch.RecordHit(tenantID, bucketTS, 1, sessionInc, 0)

	// 2. Record clean daily path
	cleanPath := CleanPath(rawPath)
	if cleanPath != "" {
		e.batch.RecordPath(tenantID, dayStr, cleanPath, 1)
	}

	// 3. Record clean domain-only referrer
	cleanRef := CleanReferrer(rawRef)
	if cleanRef != "" {
		e.batch.RecordReferrer(tenantID, dayStr, cleanRef, 1)
	}
}

func (e *Engine) RecordHeartbeat(tenantID string) {
	now := time.Now().In(e.tz)
	bucketTS := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, e.tz).Unix()
	// Each heartbeat represents 15s of accumulated active dwell time
	e.batch.RecordHit(tenantID, bucketTS, 0, 0, 15)
}

func CleanPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "/"
	}
	// Strip query parameters
	if idx := strings.IndexByte(raw, '?'); idx != -1 {
		raw = raw[:idx]
	}
	// Strip hash fragment
	if idx := strings.IndexByte(raw, '#'); idx != -1 {
		raw = raw[:idx]
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	if len(raw) > 256 {
		raw = raw[:256]
	}
	return raw
}

func CleanReferrer(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "10.") {
		return ""
	}
	if len(host) > 128 {
		host = host[:128]
	}
	return host
}

func (e *Engine) PruneOldData(ctx context.Context) error {
	cutoff := time.Now().In(e.tz).AddDate(0, 0, -e.retention)
	cutoffTS := cutoff.Unix()
	cutoffDay := cutoff.Format("2006-01-02")

	_, err := e.db.ExecContext(ctx, "DELETE FROM hits_hourly WHERE bucket_ts < ?", cutoffTS)
	if err != nil {
		return fmt.Errorf("prune hits_hourly: %w", err)
	}

	_, err = e.db.ExecContext(ctx, "DELETE FROM paths_daily WHERE day < ?", cutoffDay)
	if err != nil {
		return fmt.Errorf("prune paths_daily: %w", err)
	}

	_, err = e.db.ExecContext(ctx, "DELETE FROM referrers_daily WHERE day < ?", cutoffDay)
	if err != nil {
		return fmt.Errorf("prune referrers_daily: %w", err)
	}

	return nil
}

func (e *Engine) pruneLoop() {
	defer e.wg.Done()
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_ = e.PruneOldData(ctx)
			cancel()
		case <-e.stopCh:
			return
		}
	}
}

func (e *Engine) Close() {
	close(e.stopCh)
	e.wg.Wait()
}
