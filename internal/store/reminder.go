package store

import (
	"database/sql"
	"errors"
	"time"
)

const reminderColumns = `SELECT id, title, notes, next_at, rrule, enabled, last_fired_at, task_id, created_at FROM reminders`

// CreateReminder inserts a reminder and returns its id.
func (s *Store) CreateReminder(r *Reminder) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO reminders (title, notes, next_at, rrule, enabled, last_fired_at, task_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Title, r.Notes, r.NextAt.UTC().Format(time.RFC3339), r.RRule, boolInt(r.Enabled),
		fmtTime(r.LastFiredAt), nullInt(r.TaskID), nowStr(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetReminder loads a reminder by id.
func (s *Store) GetReminder(id int64) (*Reminder, error) {
	row := s.db.QueryRow(reminderColumns+` WHERE id = ?`, id)
	r, err := scanReminder(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return r, err
}

// ListReminders returns all reminders ordered by next fire time.
func (s *Store) ListReminders(enabledOnly bool) ([]Reminder, error) {
	q := reminderColumns
	if enabledOnly {
		q += ` WHERE enabled = 1`
	}
	q += ` ORDER BY next_at ASC`
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Reminder
	for rows.Next() {
		r, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// DueReminders returns enabled reminders whose next_at is at or before now.
func (s *Store) DueReminders(now time.Time) ([]Reminder, error) {
	rows, err := s.db.Query(reminderColumns+` WHERE enabled = 1 AND next_at <= ? ORDER BY next_at ASC`,
		now.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Reminder
	for rows.Next() {
		r, err := scanReminder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

// NextDueAt returns the earliest next_at among enabled reminders, including
// ones that are already overdue. The bool is false when no enabled reminder
// exists. next_at is stored as a UTC RFC3339 string, so MIN() is a valid
// chronological ordering.
func (s *Store) NextDueAt() (time.Time, bool, error) {
	var ns sql.NullString
	if err := s.db.QueryRow(`SELECT MIN(next_at) FROM reminders WHERE enabled = 1`).Scan(&ns); err != nil {
		return time.Time{}, false, err
	}
	if !ns.Valid || ns.String == "" {
		return time.Time{}, false, nil
	}
	t, err := time.Parse(time.RFC3339, ns.String)
	if err != nil {
		return time.Time{}, false, err
	}
	return t, true, nil
}

// UpdateReminderNext advances a reminder's schedule after firing.
func (s *Store) UpdateReminderNext(id int64, next time.Time) error {
	_, err := s.db.Exec(`UPDATE reminders SET next_at=?, last_fired_at=? WHERE id=?`,
		next.UTC().Format(time.RFC3339), nowStr(), id)
	return err
}

// SnoozeReminder pushes next_at forward and re-enables the reminder.
func (s *Store) SnoozeReminder(id int64, next time.Time) error {
	res, err := s.db.Exec(`UPDATE reminders SET next_at=?, enabled=1 WHERE id=?`,
		next.UTC().Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	return affected(res)
}

// SetReminderEnabled toggles a reminder.
func (s *Store) SetReminderEnabled(id int64, enabled bool) error {
	res, err := s.db.Exec(`UPDATE reminders SET enabled=? WHERE id=?`, boolInt(enabled), id)
	if err != nil {
		return err
	}
	return affected(res)
}

// DeleteReminder removes a reminder.
func (s *Store) DeleteReminder(id int64) error {
	res, err := s.db.Exec(`DELETE FROM reminders WHERE id=?`, id)
	if err != nil {
		return err
	}
	return affected(res)
}

func scanReminder(sc scanner) (*Reminder, error) {
	var (
		r           Reminder
		nextAt      string
		enabled     int
		lastFiredAt sql.NullString
		taskID      sql.NullInt64
		createdAt   string
	)
	if err := sc.Scan(&r.ID, &r.Title, &r.Notes, &nextAt, &r.RRule, &enabled, &lastFiredAt, &taskID, &createdAt); err != nil {
		return nil, err
	}
	r.NextAt, _ = time.Parse(time.RFC3339, nextAt)
	r.Enabled = enabled == 1
	r.LastFiredAt = parseTime(lastFiredAt)
	if taskID.Valid {
		r.TaskID = &taskID.Int64
	}
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &r, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
