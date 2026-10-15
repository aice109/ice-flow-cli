package store

import (
	"database/sql"
	"errors"
	"time"
)

const snippetColumns = `SELECT id, command, description, tags, example, shell, use_count, last_used_at, created_at FROM snippets`

// CreateSnippet inserts a command snippet and returns its id.
func (s *Store) CreateSnippet(sn *Snippet) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO snippets (command, description, tags, example, shell, use_count, last_used_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sn.Command, sn.Description, JoinTags(sn.Tags), sn.Example, sn.Shell, sn.UseCount, fmtTime(sn.LastUsedAt), nowStr(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetSnippet loads a snippet by id.
func (s *Store) GetSnippet(id int64) (*Snippet, error) {
	row := s.db.QueryRow(snippetColumns+` WHERE id = ?`, id)
	sn, err := scanSnippet(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return sn, err
}

// ListSnippets returns every snippet ordered by most recently created.
func (s *Store) ListSnippets() ([]Snippet, error) {
	rows, err := s.db.Query(snippetColumns + ` ORDER BY use_count DESC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Snippet
	for rows.Next() {
		sn, err := scanSnippet(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *sn)
	}
	return out, rows.Err()
}

// UpdateSnippet persists all mutable fields of sn.
func (s *Store) UpdateSnippet(sn *Snippet) error {
	res, err := s.db.Exec(`
		UPDATE snippets SET command=?, description=?, tags=?, example=?, shell=? WHERE id=?`,
		sn.Command, sn.Description, JoinTags(sn.Tags), sn.Example, sn.Shell, sn.ID,
	)
	if err != nil {
		return err
	}
	return affected(res)
}

// DeleteSnippet removes a snippet.
func (s *Store) DeleteSnippet(id int64) error {
	res, err := s.db.Exec(`DELETE FROM snippets WHERE id=?`, id)
	if err != nil {
		return err
	}
	return affected(res)
}

// RecordSnippetUse bumps usage counters, feeding search ranking.
func (s *Store) RecordSnippetUse(id int64) error {
	_, err := s.db.Exec(`UPDATE snippets SET use_count = use_count + 1, last_used_at = ? WHERE id=?`,
		nowStr(), id)
	return err
}

func scanSnippet(sc scanner) (*Snippet, error) {
	var (
		sn         Snippet
		tags       string
		lastUsedAt sql.NullString
		createdAt  string
	)
	if err := sc.Scan(&sn.ID, &sn.Command, &sn.Description, &tags, &sn.Example, &sn.Shell, &sn.UseCount, &lastUsedAt, &createdAt); err != nil {
		return nil, err
	}
	sn.Tags = SplitTags(tags)
	sn.LastUsedAt = parseTime(lastUsedAt)
	sn.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &sn, nil
}
