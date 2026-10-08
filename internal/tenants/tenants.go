package tenants

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ramhandean/diorama-ops/internal/db"
)

var (
	ErrNotFound       = errors.New("tenant not found")
	ErrDuplicateKey   = errors.New("site key or tenant id already exists")
	ErrOriginRejected = errors.New("origin not allowed for this tenant")
)

type Tenant struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Accent      string    `json:"accent"`
	LogoPath    string    `json:"logo_path,omitempty"`
	UseFavicon  bool      `json:"use_favicon"`
	SiteKey     string    `json:"site_key"`
	Origins     []string  `json:"origins"`
	PositionX   *float64  `json:"position_x,omitempty"`
	PositionY   *float64  `json:"position_y,omitempty"`
	PositionZ   *float64  `json:"position_z,omitempty"`
	KumaURL     string    `json:"kuma_url,omitempty"`
	HealthURL   string    `json:"health_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Store struct {
	db *db.DB
}

func NewStore(db *db.DB) *Store {
	return &Store{db: db}
}

func GenerateSiteKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "pk_" + hex.EncodeToString(b)
}

func (s *Store) List(ctx context.Context) ([]Tenant, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, url, accent, logo_path, use_favicon, site_key, origins,
		       position_x, position_y, position_z, kuma_url, health_url, created_at
		FROM tenants
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	list := make([]Tenant, 0)
	for rows.Next() {
		var t Tenant
		var originsJSON string
		var useFaviconInt int
		var createdAtUnix int64
		var posX, posY, posZ sql.NullFloat64
		var logoPath, kumaURL, healthURL sql.NullString

		err := rows.Scan(
			&t.ID, &t.Name, &t.URL, &t.Accent, &logoPath, &useFaviconInt, &t.SiteKey, &originsJSON,
			&posX, &posY, &posZ, &kumaURL, &healthURL, &createdAtUnix,
		)
		if err != nil {
			return nil, fmt.Errorf("scan tenant row: %w", err)
		}

		t.UseFavicon = useFaviconInt == 1
		t.CreatedAt = time.Unix(createdAtUnix, 0)
		if logoPath.Valid {
			t.LogoPath = logoPath.String
		}
		if kumaURL.Valid {
			t.KumaURL = kumaURL.String
		}
		if healthURL.Valid {
			t.HealthURL = healthURL.String
		}
		if posX.Valid {
			t.PositionX = &posX.Float64
		}
		if posY.Valid {
			t.PositionY = &posY.Float64
		}
		if posZ.Valid {
			t.PositionZ = &posZ.Float64
		}

		_ = json.Unmarshal([]byte(originsJSON), &t.Origins)
		if t.Origins == nil {
			t.Origins = []string{}
		}

		list = append(list, t)
	}

	return list, nil
}

func (s *Store) GetByID(ctx context.Context, id string) (*Tenant, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, url, accent, logo_path, use_favicon, site_key, origins,
		       position_x, position_y, position_z, kuma_url, health_url, created_at
		FROM tenants
		WHERE id = ?
	`, id)
	return scanTenant(row)
}

func (s *Store) GetBySiteKey(ctx context.Context, siteKey string) (*Tenant, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, url, accent, logo_path, use_favicon, site_key, origins,
		       position_x, position_y, position_z, kuma_url, health_url, created_at
		FROM tenants
		WHERE site_key = ?
	`, siteKey)
	return scanTenant(row)
}

func (s *Store) Create(ctx context.Context, t *Tenant) error {
	if t.ID == "" {
		return errors.New("tenant id (slug) is required")
	}
	if t.SiteKey == "" {
		t.SiteKey = GenerateSiteKey()
	}
	if t.Accent == "" {
		t.Accent = "#3b82f6"
	}
	if t.Origins == nil {
		t.Origins = []string{}
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}

	originsBytes, _ := json.Marshal(t.Origins)
	useFaviconInt := 0
	if t.UseFavicon {
		useFaviconInt = 1
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tenants (
			id, name, url, accent, logo_path, use_favicon, site_key, origins,
			position_x, position_y, position_z, kuma_url, health_url, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		t.ID, t.Name, t.URL, t.Accent, nullString(t.LogoPath), useFaviconInt, t.SiteKey, string(originsBytes),
		nullFloat(t.PositionX), nullFloat(t.PositionY), nullFloat(t.PositionZ),
		nullString(t.KumaURL), nullString(t.HealthURL), t.CreatedAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("insert tenant: %w", err)
	}
	return nil
}

func (s *Store) Update(ctx context.Context, t *Tenant) error {
	originsBytes, _ := json.Marshal(t.Origins)
	useFaviconInt := 0
	if t.UseFavicon {
		useFaviconInt = 1
	}

	res, err := s.db.ExecContext(ctx, `
		UPDATE tenants SET
			name = ?, url = ?, accent = ?, logo_path = ?, use_favicon = ?,
			origins = ?, position_x = ?, position_y = ?, position_z = ?,
			kuma_url = ?, health_url = ?
		WHERE id = ?
	`,
		t.Name, t.URL, t.Accent, nullString(t.LogoPath), useFaviconInt,
		string(originsBytes), nullFloat(t.PositionX), nullFloat(t.PositionY), nullFloat(t.PositionZ),
		nullString(t.KumaURL), nullString(t.HealthURL), t.ID,
	)
	if err != nil {
		return fmt.Errorf("update tenant: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, "DELETE FROM tenants WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (t *Tenant) ValidateOrigin(rawOrigin string) bool {
	if rawOrigin == "" {
		return false
	}
	normOrigin := normalizeHost(rawOrigin)
	for _, allowed := range t.Origins {
		normAllowed := normalizeHost(allowed)
		if normAllowed == "*" || strings.EqualFold(normAllowed, normOrigin) {
			return true
		}
	}
	return false
}

func normalizeHost(raw string) string {
	raw = strings.TrimSpace(raw)
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return strings.ToLower(raw)
	}
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port != "" && port != "80" && port != "443" {
		host = host + ":" + port
	}
	return host
}

func scanTenant(row *sql.Row) (*Tenant, error) {
	var t Tenant
	var originsJSON string
	var useFaviconInt int
	var createdAtUnix int64
	var posX, posY, posZ sql.NullFloat64
	var logoPath, kumaURL, healthURL sql.NullString

	err := row.Scan(
		&t.ID, &t.Name, &t.URL, &t.Accent, &logoPath, &useFaviconInt, &t.SiteKey, &originsJSON,
		&posX, &posY, &posZ, &kumaURL, &healthURL, &createdAtUnix,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	t.UseFavicon = useFaviconInt == 1
	t.CreatedAt = time.Unix(createdAtUnix, 0)
	if logoPath.Valid {
		t.LogoPath = logoPath.String
	}
	if kumaURL.Valid {
		t.KumaURL = kumaURL.String
	}
	if healthURL.Valid {
		t.HealthURL = healthURL.String
	}
	if posX.Valid {
		t.PositionX = &posX.Float64
	}
	if posY.Valid {
		t.PositionY = &posY.Float64
	}
	if posZ.Valid {
		t.PositionZ = &posZ.Float64
	}

	_ = json.Unmarshal([]byte(originsJSON), &t.Origins)
	if t.Origins == nil {
		t.Origins = []string{}
	}

	return &t, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullFloat(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{Valid: false}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}
