package tenants

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ramhandean/diorama-ops/internal/db"
)

func TestOriginValidation(t *testing.T) {
	tenant := &Tenant{
		ID:      "test-tenant",
		Origins: []string{"https://myblog.com", "http://localhost:3000", "*.example.org"},
	}

	tests := []struct {
		origin string
		allow  bool
	}{
		{"https://myblog.com", true},
		{"http://myblog.com", true}, // host match
		{"https://myblog.com:443", true},
		{"http://localhost:3000", true},
		{"http://localhost:8080", false},
		{"https://evil.com", false},
		{"", false},
	}

	for _, tc := range tests {
		got := tenant.ValidateOrigin(tc.origin)
		if got != tc.allow {
			t.Errorf("origin %q: expected allow=%v, got=%v", tc.origin, tc.allow, got)
		}
	}
}

func TestTenantCRUD(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dioramaops-tenants-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dbConn, err := db.Open(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	defer dbConn.Close()

	store := NewStore(dbConn)
	ctx := context.Background()

	// 1. Create
	newT := &Tenant{
		ID:         "blog",
		Name:       "Personal Blog",
		URL:        "https://blog.example.com",
		Accent:     "#10b981",
		UseFavicon: true,
		Origins:    []string{"https://blog.example.com"},
		CreatedAt:  time.Now(),
	}

	if err := store.Create(ctx, newT); err != nil {
		t.Fatalf("create tenant: %v", err)
	}

	// 2. Get by ID
	fetched, err := store.GetByID(ctx, "blog")
	if err != nil {
		t.Fatalf("get tenant: %v", err)
	}
	if fetched.Name != "Personal Blog" || fetched.SiteKey == "" {
		t.Fatalf("unexpected tenant data: %+v", fetched)
	}

	// 3. Get by SiteKey
	byKey, err := store.GetBySiteKey(ctx, fetched.SiteKey)
	if err != nil {
		t.Fatalf("get by site key: %v", err)
	}
	if byKey.ID != "blog" {
		t.Fatalf("expected id blog, got %s", byKey.ID)
	}

	// 4. Update
	fetched.Name = "Updated Blog"
	if err := store.Update(ctx, fetched); err != nil {
		t.Fatalf("update tenant: %v", err)
	}

	updated, _ := store.GetByID(ctx, "blog")
	if updated.Name != "Updated Blog" {
		t.Fatalf("expected updated name, got %s", updated.Name)
	}

	// 5. Delete
	if err := store.Delete(ctx, "blog"); err != nil {
		t.Fatalf("delete tenant: %v", err)
	}

	_, err = store.GetByID(ctx, "blog")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
