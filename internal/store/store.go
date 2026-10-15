// Package store is the persistence layer. It wraps a pure-Go SQLite database
// (modernc.org/sqlite, no cgo) and exposes typed accessors for tasks,
// reminders and command snippets.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // registers the pure-Go "sqlite" driver
)

// Store owns the database handle.
type Store struct {
	db   *sql.DB
	path string
}

// Open opens (creating if needed) the SQLite database at path and runs
// migrations.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// modernc's driver is safe for concurrent use, but a CLI is single-user;
	// capping the pool avoids surprise WAL contention.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db, path: path}, nil
}

// Close releases the database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Path reports the database file path.
func (s *Store) Path() string { return s.path }

// --- shared helpers -------------------------------------------------------

func nowStr() string { return time.Now().UTC().Format(time.RFC3339) }

func timePtr(t time.Time) *time.Time { return &t }

func fmtTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func parseTime(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, ns.String)
	if err != nil {
		return nil
	}
	return &t
}

func nullInt(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}
