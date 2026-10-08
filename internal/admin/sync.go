package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/ramhandean/diorama-ops/internal/tenants"
)

type SyncManager struct {
	tenantsStore *tenants.Store
}

func NewSyncManager(ts *tenants.Store) *SyncManager {
	return &SyncManager{tenantsStore: ts}
}

func (sm *SyncManager) Export(ctx context.Context, w io.Writer) error {
	list, err := sm.tenantsStore.List(ctx)
	if err != nil {
		return fmt.Errorf("list tenants: %w", err)
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(list)
}

func (sm *SyncManager) Import(ctx context.Context, r io.Reader) (int, error) {
	var list []tenants.Tenant
	if err := json.NewDecoder(r).Decode(&list); err != nil {
		return 0, fmt.Errorf("decode tenants json: %w", err)
	}

	imported := 0
	for _, t := range list {
		if t.ID == "" {
			continue
		}
		if t.CreatedAt.IsZero() {
			t.CreatedAt = time.Now()
		}

		existing, err := sm.tenantsStore.GetByID(ctx, t.ID)
		if err == nil && existing != nil {
			// Update
			if t.SiteKey == "" {
				t.SiteKey = existing.SiteKey
			}
			if err := sm.tenantsStore.Update(ctx, &t); err != nil {
				return imported, fmt.Errorf("update tenant %s: %w", t.ID, err)
			}
		} else {
			// Create
			if err := sm.tenantsStore.Create(ctx, &t); err != nil {
				return imported, fmt.Errorf("create tenant %s: %w", t.ID, err)
			}
		}
		imported++
	}

	return imported, nil
}
