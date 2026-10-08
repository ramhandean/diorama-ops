package server

import (
	"net/http"
	"time"
)

type HourlyPoint struct {
	Timestamp int64  `json:"timestamp"`
	Label     string `json:"label"`
	Pageviews int    `json:"pageviews"`
	Sessions  int    `json:"sessions"`
}

type TopItem struct {
	Name string `json:"name"`
	Hits int    `json:"hits"`
}

type OverviewStats struct {
	ActiveVisitors   int           `json:"active_visitors"`
	TotalPageviews24 int           `json:"total_pageviews_24h"`
	TotalSessions24  int           `json:"total_sessions_24h"`
	AvgDwell24       int           `json:"avg_dwell_24h"`
	HourlyPoints     []HourlyPoint `json:"hourly_points"`
}

type TenantStats struct {
	TenantID     string        `json:"tenant_id"`
	Range        string        `json:"range"`
	Points       []HourlyPoint `json:"points"`
	TopPaths     []TopItem     `json:"top_paths"`
	TopReferrers []TopItem     `json:"top_referrers"`
}

func (s *Server) handleStatsOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	now := time.Now().In(s.cfg.TZ)
	since24h := now.Add(-24 * time.Hour).Unix()

	var totalPV, totalSess, totalDwell int
	row := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(pageviews), 0), COALESCE(SUM(sessions), 0), COALESCE(SUM(dwell_sum), 0)
		FROM hits_hourly
		WHERE bucket_ts >= ?
	`, since24h)
	_ = row.Scan(&totalPV, &totalSess, &totalDwell)

	avgDwell := 0
	if totalSess > 0 {
		avgDwell = totalDwell / totalSess
	}

	// Hourly breakdown last 24h
	rows, err := s.db.QueryContext(ctx, `
		SELECT bucket_ts, SUM(pageviews), SUM(sessions)
		FROM hits_hourly
		WHERE bucket_ts >= ?
		GROUP BY bucket_ts
		ORDER BY bucket_ts ASC
	`, since24h)

	var points []HourlyPoint
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var bts int64
			var pv, sess int
			if err := rows.Scan(&bts, &pv, &sess); err == nil {
				label := time.Unix(bts, 0).In(s.cfg.TZ).Format("15:04")
				points = append(points, HourlyPoint{
					Timestamp: bts,
					Label:     label,
					Pageviews: pv,
					Sessions:  sess,
				})
			}
		}
	}
	if points == nil {
		points = []HourlyPoint{}
	}

	res := OverviewStats{
		ActiveVisitors:   s.presence.TotalActiveVisitors(),
		TotalPageviews24: totalPV,
		TotalSessions24:  totalSess,
		AvgDwell24:       avgDwell,
		HourlyPoints:     points,
	}

	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleTenantStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := r.PathValue("id")
	rangeParam := r.URL.Query().Get("range")
	if rangeParam != "7d" {
		rangeParam = "24h"
	}

	now := time.Now().In(s.cfg.TZ)
	var sinceTS int64
	var sinceDay string

	if rangeParam == "7d" {
		sinceTS = now.Add(-7 * 24 * time.Hour).Unix()
		sinceDay = now.Add(-7 * 24 * time.Hour).Format("2006-01-02")
	} else {
		sinceTS = now.Add(-24 * time.Hour).Unix()
		sinceDay = now.Format("2006-01-02")
	}

	// 1. Hourly/Daily trend points
	rows, err := s.db.QueryContext(ctx, `
		SELECT bucket_ts, SUM(pageviews), SUM(sessions)
		FROM hits_hourly
		WHERE tenant_id = ? AND bucket_ts >= ?
		GROUP BY bucket_ts
		ORDER BY bucket_ts ASC
	`, tenantID, sinceTS)

	var points []HourlyPoint
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var bts int64
			var pv, sess int
			if err := rows.Scan(&bts, &pv, &sess); err == nil {
				format := "15:04"
				if rangeParam == "7d" {
					format = "02 Jan 15:04"
				}
				label := time.Unix(bts, 0).In(s.cfg.TZ).Format(format)
				points = append(points, HourlyPoint{
					Timestamp: bts,
					Label:     label,
					Pageviews: pv,
					Sessions:  sess,
				})
			}
		}
	}
	if points == nil {
		points = []HourlyPoint{}
	}

	// 2. Top paths
	var topPaths []TopItem
	pathRows, err := s.db.QueryContext(ctx, `
		SELECT path, SUM(hits) as total_hits
		FROM paths_daily
		WHERE tenant_id = ? AND day >= ?
		GROUP BY path
		ORDER BY total_hits DESC
		LIMIT 10
	`, tenantID, sinceDay)
	if err == nil {
		defer pathRows.Close()
		for pathRows.Next() {
			var name string
			var hits int
			if err := pathRows.Scan(&name, &hits); err == nil {
				topPaths = append(topPaths, TopItem{Name: name, Hits: hits})
			}
		}
	}
	if topPaths == nil {
		topPaths = []TopItem{}
	}

	// 3. Top referrers
	var topRefs []TopItem
	refRows, err := s.db.QueryContext(ctx, `
		SELECT domain, SUM(hits) as total_hits
		FROM referrers_daily
		WHERE tenant_id = ? AND day >= ?
		GROUP BY domain
		ORDER BY total_hits DESC
		LIMIT 10
	`, tenantID, sinceDay)
	if err == nil {
		defer refRows.Close()
		for refRows.Next() {
			var name string
			var hits int
			if err := refRows.Scan(&name, &hits); err == nil {
				topRefs = append(topRefs, TopItem{Name: name, Hits: hits})
			}
		}
	}
	if topRefs == nil {
		topRefs = []TopItem{}
	}

	res := TenantStats{
		TenantID:     tenantID,
		Range:        rangeParam,
		Points:       points,
		TopPaths:     topPaths,
		TopReferrers: topRefs,
	}

	writeJSON(w, http.StatusOK, res)
}
