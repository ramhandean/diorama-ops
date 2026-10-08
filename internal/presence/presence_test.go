package presence

import (
	"testing"
	"time"
)

func TestHashVID(t *testing.T) {
	tr := NewTracker(1*time.Second, time.UTC)
	defer tr.Close()

	vid1 := tr.HashVID("192.168.1.50", "Mozilla/5.0", "pk_123")
	vid2 := tr.HashVID("192.168.1.50", "Mozilla/5.0", "pk_123")
	vidDifferentIP := tr.HashVID("192.168.1.51", "Mozilla/5.0", "pk_123")

	if vid1 == "" {
		t.Fatal("expected non-empty vid")
	}
	if vid1 != vid2 {
		t.Fatalf("expected identical hash for same inputs, got %s and %s", vid1, vid2)
	}
	if vid1 == vidDifferentIP {
		t.Fatalf("expected different hash for different IP")
	}
}

func TestPresenceDeduplicationAndTTL(t *testing.T) {
	tr := NewTracker(500*time.Millisecond, time.UTC)
	defer tr.Close()

	ch := tr.Subscribe()
	defer tr.Unsubscribe(ch)

	vid := "v_testuser"
	tid := "tenant-a"

	// 1. First tab (sid1) opens
	isNewVID, isNewSID := tr.Touch("tab-1", vid, tid, "/", "")
	if !isNewVID || !isNewSID {
		t.Fatalf("expected new VID and SID on first touch")
	}

	select {
	case ev := <-ch:
		if ev.Type != EventEnter || ev.TenantID != tid || ev.VID != vid {
			t.Fatalf("unexpected event: %+v", ev)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for enter event")
	}

	if count := tr.GetTenantActiveCount(tid); count != 1 {
		t.Fatalf("expected 1 active visitor, got %d", count)
	}

	// 2. Second tab (sid2) opens by SAME user (vid)
	isNewVID, isNewSID = tr.Touch("tab-2", vid, tid, "/docs", "")
	if isNewVID {
		t.Fatalf("same VID should NOT trigger isNewVID=true")
	}
	if !isNewSID {
		t.Fatalf("tab-2 is a new SID")
	}

	// Active visitor count remains 1 because it's the same person
	if count := tr.GetTenantActiveCount(tid); count != 1 {
		t.Fatalf("expected 1 active visitor (deduped), got %d", count)
	}

	// 3. Close tab-1
	tr.Remove("tab-1")
	// Count still 1 because tab-2 is still open
	if count := tr.GetTenantActiveCount(tid); count != 1 {
		t.Fatalf("expected 1 active visitor remaining, got %d", count)
	}

	// 4. Close tab-2
	tr.Remove("tab-2")
	select {
	case ev := <-ch:
		if ev.Type != EventLeave || ev.TenantID != tid || ev.VID != vid {
			t.Fatalf("unexpected leave event: %+v", ev)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for leave event")
	}

	if count := tr.GetTenantActiveCount(tid); count != 0 {
		t.Fatalf("expected 0 active visitors after all tabs closed, got %d", count)
	}
}
