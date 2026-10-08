-- 001_init.sql: DioramaOps foundational database schema

CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS tenants (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  url TEXT NOT NULL,
  accent TEXT NOT NULL DEFAULT '#3b82f6',
  logo_path TEXT,
  use_favicon INTEGER NOT NULL DEFAULT 1,
  site_key TEXT NOT NULL UNIQUE,
  origins TEXT NOT NULL DEFAULT '[]',
  position_x REAL,
  position_y REAL,
  position_z REAL,
  kuma_url TEXT,
  health_url TEXT,
  created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tenants_site_key ON tenants(site_key);

CREATE TABLE IF NOT EXISTS hits_hourly (
  tenant_id TEXT NOT NULL,
  bucket_ts INTEGER NOT NULL,
  pageviews INTEGER NOT NULL DEFAULT 0,
  sessions INTEGER NOT NULL DEFAULT 0,
  dwell_sum INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (tenant_id, bucket_ts),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS paths_daily (
  tenant_id TEXT NOT NULL,
  day TEXT NOT NULL,
  path TEXT NOT NULL,
  hits INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (tenant_id, day, path),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS referrers_daily (
  tenant_id TEXT NOT NULL,
  day TEXT NOT NULL,
  domain TEXT NOT NULL,
  hits INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (tenant_id, day, domain),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
