package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ramhandean/diorama-ops/internal/db/migrations"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
	path string
}

func Open(dataDir string) (*DB, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir %s: %w", dataDir, err)
	}

	dbPath := filepath.Join(dataDir, "dioramaops.db")
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)", dbPath)

	dbConn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	// SQLite WAL handles concurrent readers well, limit connections for optimal WAL performance
	dbConn.SetMaxOpenConns(10)
	dbConn.SetMaxIdleConns(5)
	dbConn.SetConnMaxLifetime(time.Hour)

	db := &DB{DB: dbConn, path: dbPath}
	if err := db.Migrate(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	return db, nil
}

func (d *DB) Migrate(ctx context.Context) error {
	_, err := d.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at INTEGER NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	entries, err := migrations.FS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read migrations embedded fs: %w", err)
	}

	type migrationFile struct {
		version int
		name    string
	}
	var migFiles []migrationFile

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		parts := strings.Split(entry.Name(), "_")
		if len(parts) < 2 {
			continue
		}
		ver, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		migFiles = append(migFiles, migrationFile{version: ver, name: entry.Name()})
	}

	sort.Slice(migFiles, func(i, j int) bool {
		return migFiles[i].version < migFiles[j].version
	})

	for _, mf := range migFiles {
		var count int
		err := d.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = ?", mf.version).Scan(&count)
		if err != nil {
			return fmt.Errorf("check migration version %d: %w", mf.version, err)
		}
		if count > 0 {
			continue
		}

		content, err := migrations.FS.ReadFile(mf.name)
		if err != nil {
			return fmt.Errorf("read migration file %s: %w", mf.name, err)
		}

		tx, err := d.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx for migration %d: %w", mf.version, err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", mf.name, err)
		}

		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)", mf.version, time.Now().Unix()); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %d: %w", mf.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", mf.version, err)
		}
	}

	return nil
}

func (d *DB) Backup(ctx context.Context, backupDir string) (string, error) {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("create backup dir: %w", err)
	}
	destFile := filepath.Join(backupDir, fmt.Sprintf("backup-%s.db", time.Now().Format("20060102-150405")))
	query := fmt.Sprintf("VACUUM INTO '%s'", destFile)
	if _, err := d.ExecContext(ctx, query); err != nil {
		return "", fmt.Errorf("vacuum into %s: %w", destFile, err)
	}
	return destFile, nil
}
