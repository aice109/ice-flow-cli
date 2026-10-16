package cli

import (
	"time"

	"github.com/aice109/ice-flow-cli/internal/store"
)

// The database stores timestamps in UTC, which is correct for storage but
// confusing in --json output: a task entered as "09:00" in UTC+8 came back as
// "01:00Z". These view types render every timestamp in the local timezone as
// RFC3339 with an explicit offset (e.g. 2026-10-16T09:00:00+08:00), so the
// value matches what the user typed and stays unambiguous for scripts.
//
// Field names mirror the store models on purpose, so the JSON keys are
// unchanged.

func jsonTimeValue(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format(time.RFC3339)
}

func jsonTimePtr(t *time.Time) *string {
	if t == nil || t.IsZero() {
		return nil
	}
	s := t.Local().Format(time.RFC3339)
	return &s
}

type taskJSON struct {
	ID          int64
	Title       string
	Notes       string
	Status      string
	Priority    int
	Project     string
	ParentID    *int64
	DueAt       *string
	AllDay      bool
	Tags        []string
	CreatedAt   string
	UpdatedAt   string
	CompletedAt *string
}

func newTaskJSON(t store.Task) taskJSON {
	return taskJSON{
		ID:          t.ID,
		Title:       t.Title,
		Notes:       t.Notes,
		Status:      t.Status,
		Priority:    t.Priority,
		Project:     t.Project,
		ParentID:    t.ParentID,
		DueAt:       jsonTimePtr(t.DueAt),
		AllDay:      t.AllDay,
		Tags:        t.Tags,
		CreatedAt:   jsonTimeValue(t.CreatedAt),
		UpdatedAt:   jsonTimeValue(t.UpdatedAt),
		CompletedAt: jsonTimePtr(t.CompletedAt),
	}
}

func tasksJSON(ts []store.Task) []taskJSON {
	out := make([]taskJSON, 0, len(ts))
	for _, t := range ts {
		out = append(out, newTaskJSON(t))
	}
	return out
}

type reminderJSON struct {
	ID          int64
	Title       string
	Notes       string
	NextAt      string
	RRule       string
	Enabled     bool
	LastFiredAt *string
	TaskID      *int64
	CreatedAt   string
}

func newReminderJSON(r store.Reminder) reminderJSON {
	return reminderJSON{
		ID:          r.ID,
		Title:       r.Title,
		Notes:       r.Notes,
		NextAt:      jsonTimeValue(r.NextAt),
		RRule:       r.RRule,
		Enabled:     r.Enabled,
		LastFiredAt: jsonTimePtr(r.LastFiredAt),
		TaskID:      r.TaskID,
		CreatedAt:   jsonTimeValue(r.CreatedAt),
	}
}

func remindersJSON(rs []store.Reminder) []reminderJSON {
	out := make([]reminderJSON, 0, len(rs))
	for _, r := range rs {
		out = append(out, newReminderJSON(r))
	}
	return out
}

type snippetJSON struct {
	ID          int64
	Command     string
	Description string
	Tags        []string
	Example     string
	Shell       string
	UseCount    int
	LastUsedAt  *string
	CreatedAt   string
}

func newSnippetJSON(sn store.Snippet) snippetJSON {
	return snippetJSON{
		ID:          sn.ID,
		Command:     sn.Command,
		Description: sn.Description,
		Tags:        sn.Tags,
		Example:     sn.Example,
		Shell:       sn.Shell,
		UseCount:    sn.UseCount,
		LastUsedAt:  jsonTimePtr(sn.LastUsedAt),
		CreatedAt:   jsonTimeValue(sn.CreatedAt),
	}
}

func snippetsJSON(sns []store.Snippet) []snippetJSON {
	out := make([]snippetJSON, 0, len(sns))
	for _, sn := range sns {
		out = append(out, newSnippetJSON(sn))
	}
	return out
}

type matchJSON struct {
	ID          int64
	Command     string
	Description string
	Tags        []string
	Example     string
	Shell       string
	UseCount    int
	Score       int
}

func matchesJSON(ms []scoredSnippet) []matchJSON {
	out := make([]matchJSON, 0, len(ms))
	for _, m := range ms {
		out = append(out, matchJSON{
			ID:          m.Snippet.ID,
			Command:     m.Snippet.Command,
			Description: m.Snippet.Description,
			Tags:        m.Snippet.Tags,
			Example:     m.Snippet.Example,
			Shell:       m.Snippet.Shell,
			UseCount:    m.Snippet.UseCount,
			Score:       m.Score,
		})
	}
	return out
}
