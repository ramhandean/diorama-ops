package favicon

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
)

func TestIsPrivateOrLocal(t *testing.T) {
	tests := []struct {
		ip      string
		blocked bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.5", true},
		{"172.16.0.1", true},
		{"192.168.1.100", true},
		{"169.254.169.254", true}, // AWS metadata
		{"::1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}

	for _, tc := range tests {
		ip := net.ParseIP(tc.ip)
		if got := isPrivateOrLocal(ip); got != tc.blocked {
			t.Errorf("ip %s: expected blocked=%v, got=%v", tc.ip, tc.blocked, got)
		}
	}
}

func TestSSRFBlocked(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dioramaops-fav-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	svc := NewService(tmpDir)
	_, _, err = svc.FetchFavicon(context.Background(), "http://127.0.0.1:8080/favicon.ico")
	if err == nil {
		t.Fatal("expected SSRF error for 127.0.0.1, got nil")
	}
	if !errors.Is(err, ErrPrivateIP) {
		t.Fatalf("expected ErrPrivateIP, got: %v", err)
	}
}
