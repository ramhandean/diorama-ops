package uptime

import (
	"sync"
	"testing"
)

type mockBroadcaster struct {
	mu     sync.Mutex
	events []string
}

func (m *mockBroadcaster) BroadcastStatus(tenantID, state string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, tenantID+":"+state)
}

func TestHysteresisTransitions(t *testing.T) {
	mb := &mockBroadcaster{}
	engine := &Engine{
		broadcaster: mb,
		statusMap:   make(map[string]*TenantStatusRecord),
	}

	tid := "tenant-test"

	// Initial observation: UP
	engine.applyStatusWithHysteresis(tid, StatusUp)
	if s := engine.GetTenantStatus(tid); s != StatusUp {
		t.Fatalf("expected initial status Up, got %s", s)
	}

	// 1 blip of DOWN: status should STAY UP due to hysteresis
	engine.applyStatusWithHysteresis(tid, StatusDown)
	if s := engine.GetTenantStatus(tid); s != StatusUp {
		t.Fatalf("single blip should not flip status: expected Up, got %s", s)
	}

	// Second consecutive observation of DOWN: should transition to DOWN
	engine.applyStatusWithHysteresis(tid, StatusDown)
	if s := engine.GetTenantStatus(tid); s != StatusDown {
		t.Fatalf("two consecutive downs should flip status: expected Down, got %s", s)
	}

	// 1 blip of UP: status should STAY DOWN
	engine.applyStatusWithHysteresis(tid, StatusUp)
	if s := engine.GetTenantStatus(tid); s != StatusDown {
		t.Fatalf("single recovery blip should not flip status: expected Down, got %s", s)
	}

	// Second consecutive observation of UP: should transition to UP
	engine.applyStatusWithHysteresis(tid, StatusUp)
	if s := engine.GetTenantStatus(tid); s != StatusUp {
		t.Fatalf("two consecutive ups should flip status: expected Up, got %s", s)
	}
}
