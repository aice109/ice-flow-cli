package store

import (
	"database/sql"
	"fmt"
)

// migrations is an ordered list of schema versions. Each entry is applied
// exactly once; the current version is tracked in schema_meta.
var migrations = []string{
	// v1: initial schema.
	`
CREATE TABLE tasks (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	title        TEXT    NOT NULL,
	notes        TEXT    NOT NULL DEFAULT '',
	status       TEXT    NOT NULL DEFAULT 'todo',
	priority     INTEGER NOT NULL DEFAULT 2,
	project      TEXT    NOT NULL DEFAULT '',
	parent_id    INTEGER REFERENCES tasks(id) ON DELETE CASCADE,
	due_at       TEXT,
	tags         TEXT    NOT NULL DEFAULT '',
	created_at   TEXT    NOT NULL,
	updated_at   TEXT    NOT NULL,
	completed_at TEXT
);

CREATE TABLE reminders (
	id            INTEGER PRIMARY KEY AUTOINCREMENT,
	title         TEXT    NOT NULL,
	notes         TEXT    NOT NULL DEFAULT '',
	next_at       TEXT    NOT NULL,
	rrule         TEXT    NOT NULL DEFAULT '',
	enabled       INTEGER NOT NULL DEFAULT 1,
	last_fired_at TEXT,
	task_id       INTEGER REFERENCES tasks(id) ON DELETE SET NULL,
	created_at    TEXT    NOT NULL
);

CREATE TABLE snippets (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	command      TEXT    NOT NULL,
	description  TEXT    NOT NULL DEFAULT '',
	tags         TEXT    NOT NULL DEFAULT '',
	example      TEXT    NOT NULL DEFAULT '',
	shell        TEXT    NOT NULL DEFAULT '',
	use_count    INTEGER NOT NULL DEFAULT 0,
	last_used_at TEXT,
	created_at   TEXT    NOT NULL
);

CREATE INDEX idx_tasks_status_due ON tasks(status, due_at);
CREATE INDEX idx_tasks_parent ON tasks(parent_id);
CREATE INDEX idx_reminders_next ON reminders(enabled, next_at);
CREATE INDEX idx_snippets_command ON snippets(command);
`,
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_meta (version INTEGER NOT NULL)`); err != nil {
		return err
	}
	var version int
	err := db.QueryRow(`SELECT version FROM schema_meta LIMIT 1`).Scan(&version)
	if err == sql.ErrNoRows {
		if _, err := db.Exec(`INSERT INTO schema_meta (version) VALUES (0)`); err != nil {
			return err
		}
		version = 0
	} else if err != nil {
		return err
	}

	for i := version; i < len(migrations); i++ {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration v%d: %w", i+1, err)
		}
		if _, err := tx.Exec(`UPDATE schema_meta SET version = ?`, i+1); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
