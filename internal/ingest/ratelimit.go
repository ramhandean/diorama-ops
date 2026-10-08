package ingest

import (
	"sync"
	"time"
)

type Limiter struct {
	mu      sync.Mutex
	limits  map[string]*bucket
	rate    float64 // tokens per sec
	burst   float64
	cleanCh chan struct{}
}

type bucket struct {
	tokens   float64
	lastSeen time.Time
}

func NewLimiter(rate, burst float64) *Limiter {
	l := &Limiter{
		limits:  make(map[string]*bucket),
		rate:    rate,
		burst:   burst,
		cleanCh: make(chan struct{}),
	}
	go l.cleanupLoop()
	return l
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.limits[key]
	if !ok {
		l.limits[key] = &bucket{
			tokens:   l.burst - 1,
			lastSeen: now,
		}
		return true
	}

	elapsed := now.Sub(b.lastSeen).Seconds()
	b.lastSeen = now
	b.tokens += elapsed * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}
	return false
}

func (l *Limiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.mu.Lock()
			now := time.Now()
			for k, b := range l.limits {
				if now.Sub(b.lastSeen) > 10*time.Minute {
					delete(l.limits, k)
				}
			}
			l.mu.Unlock()
		case <-l.cleanCh:
			return
		}
	}
}

func (l *Limiter) Close() {
	close(l.cleanCh)
}
