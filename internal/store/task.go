package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// TaskFilter narrows ListTasks results.
type TaskFilter struct {
	Status   string // "" = any; use StatusDone etc.
	Tag      string
	Project  string
	OnlyRoot bool // exclude subtasks
	ParentID *int64
}

// CreateTask inserts a task and returns its id.
func (s *Store) CreateTask(t *Task) (int64, error) {
	if t.Status == "" {
		t.Status = StatusTodo
	}
	now := nowStr()
	res, err := s.db.Exec(`
		INSERT INTO tasks (title, notes, status, priority, project, parent_id, due_at, tags, created_at, updated_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.Title, t.Notes, t.Status, t.Priority, t.Project, nullInt(t.ParentID), fmtTime(t.DueAt), JoinTags(t.Tags), now, now, fmtTime(t.CompletedAt),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetTask loads a single task by id.
func (s *Store) GetTask(id int64) (*Task, error) {
	row := s.db.QueryRow(taskColumns+` WHERE id = ?`, id)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

const taskColumns = `SELECT id, title, notes, status, priority, project, parent_id, due_at, tags, created_at, updated_at, completed_at FROM tasks`

// ListTasks returns tasks matching f, ordered by status, priority then due.
func (s *Store) ListTasks(f TaskFilter) ([]Task, error) {
	var (
		where []string
		args  []any
	)
	if f.Status != "" {
		where = append(where, "status = ?")
		args = append(args, f.Status)
	}
	if f.Project != "" {
		where = append(where, "project = ?")
		args = append(args, f.Project)
	}
	if f.Tag != "" {
		// tags is a comma-separated list; match whole tokens.
		where = append(where, "(',' || tags || ',') LIKE ?")
		args = append(args, "%,"+f.Tag+",%")
	}
	if f.ParentID != nil {
		where = append(where, "parent_id = ?")
		args = append(args, *f.ParentID)
	} else if f.OnlyRoot {
		where = append(where, "parent_id IS NULL")
	}

	q := taskColumns
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += ` ORDER BY
		CASE status WHEN 'doing' THEN 0 WHEN 'todo' THEN 1 ELSE 2 END,
		priority ASC,
		(due_at IS NULL) ASC,
		due_at ASC,
		id ASC`

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// UpdateTask persists all mutable fields of t.
func (s *Store) UpdateTask(t *Task) error {
	res, err := s.db.Exec(`
		UPDATE tasks SET title=?, notes=?, status=?, priority=?, project=?, parent_id=?, due_at=?, tags=?, updated_at=?, completed_at=?
		WHERE id=?`,
		t.Title, t.Notes, t.Status, t.Priority, t.Project, nullInt(t.ParentID), fmtTime(t.DueAt), JoinTags(t.Tags), nowStr(), fmtTime(t.CompletedAt), t.ID,
	)
	if err != nil {
		return err
	}
	return affected(res)
}

// SetTaskStatus transitions a task, stamping completed_at when done.
func (s *Store) SetTaskStatus(id int64, status string) error {
	var completed any
	if status == StatusDone {
		completed = nowStr()
	}
	res, err := s.db.Exec(`UPDATE tasks SET status=?, updated_at=?, completed_at=? WHERE id=?`,
		status, nowStr(), completed, id)
	if err != nil {
		return err
	}
	return affected(res)
}

// DeleteTask removes a task (and cascades to its subtasks).
func (s *Store) DeleteTask(id int64) error {
	res, err := s.db.Exec(`DELETE FROM tasks WHERE id=?`, id)
	if err != nil {
		return err
	}
	return affected(res)
}

// DueTasks returns open tasks whose due_at has passed.
func (s *Store) DueTasks(now time.Time) ([]Task, error) {
	rows, err := s.db.Query(taskColumns+` WHERE status != 'done' AND due_at IS NOT NULL AND due_at <= ? ORDER BY due_at ASC`,
		now.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTask(sc scanner) (*Task, error) {
	var (
		t           Task
		parentID    sql.NullInt64
		dueAt       sql.NullString
		completedAt sql.NullString
		tags        string
		createdAt   string
		updatedAt   string
	)
	if err := sc.Scan(&t.ID, &t.Title, &t.Notes, &t.Status, &t.Priority, &t.Project, &parentID,
		&dueAt, &tags, &createdAt, &updatedAt, &completedAt); err != nil {
		return nil, err
	}
	if parentID.Valid {
		t.ParentID = &parentID.Int64
	}
	t.DueAt = parseTime(dueAt)
	t.CompletedAt = parseTime(completedAt)
	t.Tags = SplitTags(tags)
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	t.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &t, nil
}

func affected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
