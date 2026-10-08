package demo

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/ramhandean/diorama-ops/internal/presence"
	"github.com/ramhandean/diorama-ops/internal/rollup"
	"github.com/ramhandean/diorama-ops/internal/tenants"
)

type Simulator struct {
	tenantsStore *tenants.Store
	presence     *presence.Tracker
	rollup       *rollup.Engine
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

func NewSimulator(ts *tenants.Store, pt *presence.Tracker, re *rollup.Engine) *Simulator {
	return &Simulator{
		tenantsStore: ts,
		presence:     pt,
		rollup:       re,
		stopCh:       make(chan struct{}),
	}
}

func (s *Simulator) Start(ctx context.Context) error {
	slog.Info("DEMO_MODE active: seeding sample tenants and starting simulated traffic...")

	sampleTenants := []tenants.Tenant{
		{
			ID:         "github",
			Name:       "Dean Ramhan (GitHub)",
			URL:        "https://github.com/ramhandean",
			Accent:     "#3b82f6",
			UseFavicon: true,
			Origins:    []string{"*"},
			CreatedAt:  time.Now(),
		},
		{
			ID:         "rantaupay",
			Name:       "RantauPay",
			URL:        "https://rantaupay.my.id",
			Accent:     "#10b981",
			UseFavicon: true,
			Origins:    []string{"*"},
			CreatedAt:  time.Now(),
		},
		{
			ID:         "engineroom",
			Name:       "Engine Room",
			URL:        "https://engineroom.my.id",
			Accent:     "#8b5cf6",
			UseFavicon: true,
			Origins:    []string{"*"},
			CreatedAt:  time.Now(),
		},
		{
			ID:         "sir-slash",
			Name:       "Sir Slash",
			URL:        "https://sir-slash.engineroom.my.id",
			Accent:     "#f59e0b",
			UseFavicon: true,
			Origins:    []string{"*"},
			CreatedAt:  time.Now(),
		},
		{
			ID:         "sahamflow",
			Name:       "SahamFlow",
			URL:        "https://sahamflow.engineroom.my.id",
			Accent:     "#06b6d4",
			UseFavicon: true,
			Origins:    []string{"*"},
			CreatedAt:  time.Now(),
		},
		{
			ID:         "diorama-ops",
			Name:       "DioramaOps Hub",
			URL:        "https://diorama-ops.engineroom.my.id",
			Accent:     "#ec4899",
			UseFavicon: true,
			Origins:    []string{"*"},
			CreatedAt:  time.Now(),
		},
	}

	for _, t := range sampleTenants {
		existing, err := s.tenantsStore.GetByID(ctx, t.ID)
		if err != nil || existing == nil {
			_ = s.tenantsStore.Create(ctx, &t)
		} else {
			t.CreatedAt = existing.CreatedAt
			_ = s.tenantsStore.Update(ctx, &t)
		}
	}

	s.wg.Add(1)
	go s.simulationLoop()
	return nil
}

func (s *Simulator) simulationLoop() {
	defer s.wg.Done()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	tenantList := []string{"github", "rantaupay", "engineroom", "sir-slash", "sahamflow", "diorama-ops"}
	samplePaths := map[string][]string{
		"github":      {"/ramhandean", "/ramhandean/diorama-ops", "/ramhandean/diorama-cloud", "/ramhandean?tab=repositories"},
		"rantaupay":   {"/", "/login", "/dashboard", "/transfer", "/docs"},
		"engineroom":  {"/", "/projects", "/status", "/services"},
		"sir-slash":   {"/", "/app", "/slash-commands", "/api"},
		"sahamflow":   {"/", "/market", "/screener", "/watchlist", "/portfolio"},
		"diorama-ops": {"/", "/analytics", "/settings", "/3d"},
	}
	sampleRefs := []string{
		"news.ycombinator.com",
		"github.com",
		"google.com",
		"twitter.com",
		"",
	}

	activeSessions := make(map[string]struct {
		tenantID string
		sid      string
		vid      string
		exitTime time.Time
	})

	ticker := time.NewTicker(1500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()

			// 1. Evict simulated sessions whose exit time has arrived
			for key, sess := range activeSessions {
				if now.After(sess.exitTime) {
					s.presence.Remove(sess.sid)
					delete(activeSessions, key)
				}
			}

			// 2. Spawn a new visitor with random probability
			if len(activeSessions) < 35 && rng.Float32() < 0.75 {
				tid := tenantList[rng.Intn(len(tenantList))]
				sid := fmt.Sprintf("demo_sid_%d_%d", now.UnixNano(), rng.Intn(10000))
				ip := fmt.Sprintf("198.51.100.%d", rng.Intn(250)+1)
				ua := "Mozilla/5.0 (DemoSimulator/1.0)"
				vid := s.presence.HashVID(ip, ua, "demo_key")

				paths := samplePaths[tid]
				path := paths[rng.Intn(len(paths))]
				ref := sampleRefs[rng.Intn(len(sampleRefs))]

				_, isNewSID := s.presence.Touch(sid, vid, tid, path, ref)
				s.rollup.RecordPageview(tid, path, ref, isNewSID)

				// Session dwells for 12 to 35 seconds
				dwellSec := 12 + rng.Intn(24)
				activeSessions[sid] = struct {
					tenantID string
					sid      string
					vid      string
					exitTime time.Time
				}{
					tenantID: tid,
					sid:      sid,
					vid:      vid,
					exitTime: now.Add(time.Duration(dwellSec) * time.Second),
				}
			}

			// 3. Send heartbeat for active sessions
			for sid, sess := range activeSessions {
				if rng.Float32() < 0.4 {
					_ = s.presence.TouchHB(sid, "")
					s.rollup.RecordHeartbeat(sess.tenantID)
				}
			}

		case <-s.stopCh:
			for _, sess := range activeSessions {
				s.presence.Remove(sess.sid)
			}
			return
		}
	}
}

func (s *Simulator) Close() {
	close(s.stopCh)
	s.wg.Wait()
}
