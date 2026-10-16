package store

import (
	"strings"
	"time"
)

// Task status values.
const (
	StatusTodo  = "todo"
	StatusDoing = "doing"
	StatusDone  = "done"
)

// Task is a single todo/plan item.
type Task struct {
	ID          int64
	Title       string
	Notes       string
	Status      string
	Priority    int // 0=P0 (highest) .. 3=P3 (lowest)
	Project     string
	ParentID    *int64
	DueAt       *time.Time
	AllDay      bool // due_at names a date, not a time of day
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
}

// PriorityLabel renders a P0..P3 label.
func PriorityLabel(p int) string {
	if p < 0 || p > 3 {
		return "P?"
	}
	return "P" + string(rune('0'+p))
}

// Reminder is a scheduled notification, optionally recurring.
type Reminder struct {
	ID          int64
	Title       string
	Notes       string
	NextAt      time.Time
	RRule       string // empty for one-shot; otherwise a FREQ=... recurrence
	Enabled     bool
	LastFiredAt *time.Time
	TaskID      *int64
	CreatedAt   time.Time
}

// Snippet is a user-defined command with an explanation.
type Snippet struct {
	ID          int64
	Command     string
	Description string
	Tags        []string
	Example     string
	Shell       string
	UseCount    int
	LastUsedAt  *time.Time
	CreatedAt   time.Time
}

// --- tag helpers ----------------------------------------------------------

// SplitTags parses a comma-separated tag list, trimming and de-duplicating.
func SplitTags(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, part := range strings.Split(s, ",") {
		t := strings.TrimSpace(part)
		if t == "" || seen[strings.ToLower(t)] {
			continue
		}
		seen[strings.ToLower(t)] = true
		out = append(out, t)
	}
	return out
}

// JoinTags renders tags back to the stored comma-separated form.
func JoinTags(tags []string) string { return strings.Join(tags, ",") }
