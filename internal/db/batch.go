package db

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type HourlyHitIncrement struct {
	TenantID  string
	BucketTS  int64
	Pageviews int
	Sessions  int
	DwellSum  int
}

type DailyPathIncrement struct {
	TenantID string
	Day      string
	Path     string
	Hits     int
}

type DailyReferrerIncrement struct {
	TenantID string
	Day      string
	Domain   string
	Hits     int
}

type BatchWriter struct {
	db        *DB
	flushSec  time.Duration
	maxBuffer int

	mu        sync.Mutex
	hits      map[string]*HourlyHitIncrement
	paths     map[string]*DailyPathIncrement
	referrers map[string]*DailyReferrerIncrement
	count     int

	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewBatchWriter(db *DB, flushInterval time.Duration, maxBuffer int) *BatchWriter {
	if flushInterval <= 0 {
		flushInterval = 5 * time.Second
	}
	if maxBuffer <= 0 {
		maxBuffer = 500
	}

	bw := &BatchWriter{
		db:        db,
		flushSec:  flushInterval,
		maxBuffer: maxBuffer,
		hits:      make(map[string]*HourlyHitIncrement),
		paths:     make(map[string]*DailyPathIncrement),
		referrers: make(map[string]*DailyReferrerIncrement),
		stopCh:    make(chan struct{}),
	}

	bw.wg.Add(1)
	go bw.loop()
	return bw
}

func (bw *BatchWriter) RecordHit(tenantID string, bucketTS int64, pageviews, sessions, dwellSum int) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	key := fmt.Sprintf("%s:%d", tenantID, bucketTS)
	if existing, found := bw.hits[key]; found {
		existing.Pageviews += pageviews
		existing.Sessions += sessions
		existing.DwellSum += dwellSum
	} else {
		bw.hits[key] = &HourlyHitIncrement{
			TenantID:  tenantID,
			BucketTS:  bucketTS,
			Pageviews: pageviews,
			Sessions:  sessions,
			DwellSum:  dwellSum,
		}
	}
	bw.count++
	if bw.count >= bw.maxBuffer {
		go bw.Flush()
	}
}

func (bw *BatchWriter) RecordPath(tenantID, day, path string, hits int) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", tenantID, day, path)
	if existing, found := bw.paths[key]; found {
		existing.Hits += hits
	} else {
		bw.paths[key] = &DailyPathIncrement{
			TenantID: tenantID,
			Day:      day,
			Path:     path,
			Hits:     hits,
		}
	}
	bw.count++
	if bw.count >= bw.maxBuffer {
		go bw.Flush()
	}
}

func (bw *BatchWriter) RecordReferrer(tenantID, day, domain string, hits int) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	key := fmt.Sprintf("%s:%s:%s", tenantID, day, domain)
	if existing, found := bw.referrers[key]; found {
		existing.Hits += hits
	} else {
		bw.referrers[key] = &DailyReferrerIncrement{
			TenantID: tenantID,
			Day:      day,
			Domain:   domain,
			Hits:     hits,
		}
	}
	bw.count++
	if bw.count >= bw.maxBuffer {
		go bw.Flush()
	}
}

func (bw *BatchWriter) Flush() error {
	bw.mu.Lock()
	if bw.count == 0 {
		bw.mu.Unlock()
		return nil
	}

	flushHits := bw.hits
	flushPaths := bw.paths
	flushRefs := bw.referrers

	bw.hits = make(map[string]*HourlyHitIncrement)
	bw.paths = make(map[string]*DailyPathIncrement)
	bw.referrers = make(map[string]*DailyReferrerIncrement)
	bw.count = 0
	bw.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := bw.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("batch begin tx: %w", err)
	}
	defer tx.Rollback()

	if len(flushHits) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO hits_hourly (tenant_id, bucket_ts, pageviews, sessions, dwell_sum)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(tenant_id, bucket_ts) DO UPDATE SET
				pageviews = pageviews + excluded.pageviews,
				sessions = sessions + excluded.sessions,
				dwell_sum = dwell_sum + excluded.dwell_sum
		`)
		if err != nil {
			return fmt.Errorf("prepare hit stmt: %w", err)
		}
		defer stmt.Close()

		for _, item := range flushHits {
			if _, err := stmt.ExecContext(ctx, item.TenantID, item.BucketTS, item.Pageviews, item.Sessions, item.DwellSum); err != nil {
				return fmt.Errorf("exec hit item: %w", err)
			}
		}
	}

	if len(flushPaths) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO paths_daily (tenant_id, day, path, hits)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(tenant_id, day, path) DO UPDATE SET
				hits = hits + excluded.hits
		`)
		if err != nil {
			return fmt.Errorf("prepare path stmt: %w", err)
		}
		defer stmt.Close()

		for _, item := range flushPaths {
			if _, err := stmt.ExecContext(ctx, item.TenantID, item.Day, item.Path, item.Hits); err != nil {
				return fmt.Errorf("exec path item: %w", err)
			}
		}
	}

	if len(flushRefs) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO referrers_daily (tenant_id, day, domain, hits)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(tenant_id, day, domain) DO UPDATE SET
				hits = hits + excluded.hits
		`)
		if err != nil {
			return fmt.Errorf("prepare referrer stmt: %w", err)
		}
		defer stmt.Close()

		for _, item := range flushRefs {
			if _, err := stmt.ExecContext(ctx, item.TenantID, item.Day, item.Domain, item.Hits); err != nil {
				return fmt.Errorf("exec referrer item: %w", err)
			}
		}
	}

	return tx.Commit()
}

func (bw *BatchWriter) loop() {
	defer bw.wg.Done()
	ticker := time.NewTicker(bw.flushSec)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = bw.Flush()
		case <-bw.stopCh:
			_ = bw.Flush()
			return
		}
	}
}

func (bw *BatchWriter) Close() error {
	close(bw.stopCh)
	bw.wg.Wait()
	return nil
}
