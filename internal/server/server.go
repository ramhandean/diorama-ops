package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ramhandean/diorama-ops/internal/admin"
	"github.com/ramhandean/diorama-ops/internal/config"
	"github.com/ramhandean/diorama-ops/internal/db"
	"github.com/ramhandean/diorama-ops/internal/demo"
	"github.com/ramhandean/diorama-ops/internal/favicon"
	"github.com/ramhandean/diorama-ops/internal/ingest"
	"github.com/ramhandean/diorama-ops/internal/metrics"
	"github.com/ramhandean/diorama-ops/internal/presence"
	"github.com/ramhandean/diorama-ops/internal/rollup"
	"github.com/ramhandean/diorama-ops/internal/tenants"
	"github.com/ramhandean/diorama-ops/internal/uptime"
)

type Server struct {
	cfg      *config.Config
	db       *db.DB
	batch    *db.BatchWriter
	tenants  *tenants.Store
	presence *presence.Tracker
	rollup   *rollup.Engine
	favicon  *favicon.Service
	uptime   *uptime.Engine
	metrics  *metrics.Registry
	logoMgr  *admin.LogoManager
	syncMgr  *admin.SyncManager
	demo     *demo.Simulator
	hub      *ingest.Hub
	server   *http.Server
}

func New(cfg *config.Config) (*Server, error) {
	dbConn, err := db.Open(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	batchWriter := db.NewBatchWriter(dbConn, 5*time.Second, 500)
	tenantStore := tenants.NewStore(dbConn)
	presenceTracker := presence.NewTracker(45*time.Second, cfg.TZ)
	rollupEngine := rollup.NewEngine(dbConn, batchWriter, cfg.TZ, cfg.RetentionDays)
	favService := favicon.NewService(filepath.Join(cfg.DataDir, "favicons"))
	hub := ingest.NewHub(cfg, tenantStore, presenceTracker, rollupEngine)
	uptimeEngine := uptime.NewEngine(tenantStore, hub)
	metricsRegistry := metrics.NewRegistry(cfg, dbConn, tenantStore, presenceTracker, uptimeEngine)
	logoManager := admin.NewLogoManager(filepath.Join(cfg.DataDir, "logos"))
	syncManager := admin.NewSyncManager(tenantStore)

	var demoSim *demo.Simulator
	if cfg.DemoMode {
		demoSim = demo.NewSimulator(tenantStore, presenceTracker, rollupEngine)
		_ = demoSim.Start(context.Background())
	}

	s := &Server{
		cfg:      cfg,
		db:       dbConn,
		batch:    batchWriter,
		tenants:  tenantStore,
		presence: presenceTracker,
		rollup:   rollupEngine,
		favicon:  favService,
		uptime:   uptimeEngine,
		metrics:  metricsRegistry,
		logoMgr:  logoManager,
		syncMgr:  syncManager,
		demo:     demoSim,
		hub:      hub,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.server = &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

func (s *Server) registerRoutes(mux *http.ServeMux) {
	// Liveness & healthcheck
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Tracker script
	mux.HandleFunc("GET /m.js", s.serveTracker)

	// Ingest & real-time
	mux.HandleFunc("GET /ws", s.hub.ServeWS)
	mux.HandleFunc("POST /api/v1/b", s.hub.ServeBeacon)
	mux.HandleFunc("OPTIONS /api/v1/b", s.hub.ServeBeacon)
	mux.HandleFunc("GET /events", s.hub.ServeSSE)

	// Public API (for 3D scene & dashboards)
	mux.HandleFunc("GET /api/v1/tenants", s.handlePublicTenants)
	mux.HandleFunc("GET /api/v1/favicon/{id}", s.handleFavicon)

	// HUD Analytics Stats API
	mux.HandleFunc("GET /api/v1/stats/overview", s.handleStatsOverview)
	mux.HandleFunc("GET /api/v1/stats/tenant/{id}", s.handleTenantStats)

	// Prometheus Metrics Exporter
	mux.HandleFunc("GET /metrics", s.metrics.ServeHTTP)

	// Uploaded logos
	logosDir := filepath.Join(s.cfg.DataDir, "logos")
	_ = os.MkdirAll(logosDir, 0755)
	mux.Handle("GET /logos/", http.StripPrefix("/logos/", http.FileServer(http.Dir(logosDir))))

	// Admin API (Bearer protected)
	mux.HandleFunc("GET /api/v1/admin/tenants", s.adminAuth(s.handleListTenants))
	mux.HandleFunc("POST /api/v1/admin/tenants", s.adminAuth(s.handleCreateTenant))
	mux.HandleFunc("GET /api/v1/admin/tenants/{id}", s.adminAuth(s.handleGetTenant))
	mux.HandleFunc("PUT /api/v1/admin/tenants/{id}", s.adminAuth(s.handleUpdateTenant))
	mux.HandleFunc("DELETE /api/v1/admin/tenants/{id}", s.adminAuth(s.handleDeleteTenant))
	mux.HandleFunc("POST /api/v1/admin/tenants/{id}/logo", s.adminAuth(s.handleUploadLogo))
	mux.HandleFunc("GET /api/v1/admin/export", s.adminAuth(s.handleExportTenants))
	mux.HandleFunc("POST /api/v1/admin/import", s.adminAuth(s.handleImportTenants))

	// Platform Configuration
	mux.HandleFunc("GET /api/v1/config", s.handleConfig)

	// Static files & frontend SPA fallback
	s.registerStatic(mux)
}

func (s *Server) serveTracker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cross-Origin-Resource-Policy", "cross-origin")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	// Look in tracker/m.js or embedded
	trackerPath := filepath.Join("tracker", "m.js")
	data, err := os.ReadFile(trackerPath)
	if err != nil {
		http.Error(w, "Tracker not found", http.StatusNotFound)
		return
	}
	_, _ = w.Write(data)
}

func (s *Server) registerStatic(mux *http.ServeMux) {
	staticDir := s.cfg.StaticDir
	if _, err := os.Stat(staticDir); os.IsNotExist(err) {
		return
	}

	fs := http.FileServer(http.Dir(staticDir))
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(staticDir, filepath.Clean(r.URL.Path))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// SPA fallback: serve index.html
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
}

func (s *Server) adminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := ""
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		if !s.cfg.ValidateAdminSecret(token) {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Server) handlePublicTenants(w http.ResponseWriter, r *http.Request) {
	list, err := s.tenants.List(r.Context())
	if err != nil {
		http.Error(w, `{"error":"failed to list tenants"}`, http.StatusInternalServerError)
		return
	}

	type PublicTenant struct {
		ID         string   `json:"id"`
		Name       string   `json:"name"`
		URL        string   `json:"url"`
		Accent     string   `json:"accent"`
		LogoPath   string   `json:"logo_path,omitempty"`
		UseFavicon bool     `json:"use_favicon"`
		PositionX  *float64 `json:"position_x,omitempty"`
		PositionY  *float64 `json:"position_y,omitempty"`
		PositionZ  *float64 `json:"position_z,omitempty"`
		Status     string   `json:"status"`
		Active     int      `json:"active"`
	}

	counts := s.presence.GetActiveCounts()
	res := make([]PublicTenant, 0, len(list))
	for _, t := range list {
		res = append(res, PublicTenant{
			ID:         t.ID,
			Name:       t.Name,
			URL:        t.URL,
			Accent:     t.Accent,
			LogoPath:   t.LogoPath,
			UseFavicon: t.UseFavicon,
			PositionX:  t.PositionX,
			PositionY:  t.PositionY,
			PositionZ:  t.PositionZ,
			Status:     string(s.uptime.GetTenantStatus(t.ID)),
			Active:     counts[t.ID],
		})
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleFavicon(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := s.tenants.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, "tenant not found", http.StatusNotFound)
		return
	}

	data, contentType, err := s.favicon.FetchFavicon(r.Context(), t.URL)
	if err != nil {
		http.Error(w, "favicon unavailable", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(data)
}

func (s *Server) handleListTenants(w http.ResponseWriter, r *http.Request) {
	list, err := s.tenants.List(r.Context())
	if err != nil {
		http.Error(w, `{"error":"failed to list tenants"}`, http.StatusInternalServerError)
		return
	}
	if list == nil {
		list = make([]tenants.Tenant, 0)
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	var t tenants.Tenant
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
		return
	}
	if err := s.tenants.Create(r.Context(), &t); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) handleGetTenant(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, err := s.tenants.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"tenant not found"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleUpdateTenant(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var t tenants.Tenant
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
		return
	}
	t.ID = id
	if err := s.tenants.Update(r.Context(), &t); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleDeleteTenant(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.tenants.Delete(r.Context(), id); err != nil {
		http.Error(w, `{"error":"failed to delete tenant"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleUploadLogo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	tenant, err := s.tenants.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"tenant not found"}`, http.StatusNotFound)
		return
	}

	// 512 KB max + multipart form headers
	if err := r.ParseMultipartForm(550 * 1024); err != nil {
		http.Error(w, `{"error":"file too large or invalid multipart form"}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("logo")
	if err != nil {
		http.Error(w, `{"error":"missing 'logo' form field"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	logoPath, err := s.logoMgr.SaveLogo(id, header)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	tenant.LogoPath = logoPath
	tenant.UseFavicon = false
	if err := s.tenants.Update(r.Context(), tenant); err != nil {
		http.Error(w, `{"error":"failed to update tenant logo path"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "uploaded",
		"logo_path": logoPath,
	})
}

func (s *Server) handleExportTenants(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=tenants.config.json")
	if err := s.syncMgr.Export(r.Context(), w); err != nil {
		http.Error(w, `{"error":"failed to export tenants"}`, http.StatusInternalServerError)
	}
}

func (s *Server) handleImportTenants(w http.ResponseWriter, r *http.Request) {
	list, err := s.tenants.List(r.Context())
	if err == nil && len(list) >= 1 {
		http.Error(w, `{"error":"Versi Self-Hosted dibatasi maksimal 1 project per instance. Instance ini sudah memiliki project."}`, http.StatusBadRequest)
		return
	}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error":"failed to read request body"}`, http.StatusBadRequest)
		return
	}
	var incoming []tenants.Tenant
	if err := json.Unmarshal(bodyBytes, &incoming); err != nil {
		http.Error(w, `{"error":"invalid json body"}`, http.StatusBadRequest)
		return
	}
	if len(incoming) > 1 {
		http.Error(w, `{"error":"Versi Self-Hosted hanya mengizinkan maksimal 1 project per instance. File import berisi lebih dari 1 project."}`, http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	count, err := s.syncMgr.Import(r.Context(), r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "imported",
		"imported": count,
	})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"public_view":   s.cfg.PublicView,
		"retention_days": s.cfg.RetentionDays,
	})
}

func (s *Server) Start() error {
	slog.Info("DioramaOps hub listening", "port", s.cfg.Port, "public_url", s.cfg.PublicURL)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.demo != nil {
		s.demo.Close()
	}
	s.uptime.Close()
	s.hub.Close()
	s.rollup.Close()
	_ = s.batch.Close()
	s.presence.Close()
	_ = s.db.Close()
	return s.server.Shutdown(ctx)
}
