package cli

import (
	"testing"
	"time"

	"github.com/aice109/ice-flow-cli/internal/store"
)

func TestValidateDays(t *testing.T) {
	for _, in := range []int{1, 3, 7, 30, 365} {
		if err := validateDays(in); err != nil {
			t.Errorf("validateDays(%d) error: %v", in, err)
		}
	}
	for _, in := range []int{0, -1, 366, 999} {
		if err := validateDays(in); err == nil {
			t.Errorf("validateDays(%d) expected an error", in)
		}
	}
}

func TestInProgressTasksExcludesListed(t *testing.T) {
	now := time.Date(2026, 10, 16, 10, 0, 0, 0, time.Local)
	today := startOfLocalDay(now)
	lastDay := today
	end := lastDay.AddDate(0, 0, 1).Add(-time.Second)

	due := time.Date(2026, 10, 16, 0, 0, 0, 0, time.Local)
	tasks := []store.Task{
		{ID: 1, Title: "listed doing", Status: store.StatusDoing, DueAt: &due, AllDay: true},
		{ID: 2, Title: "unlisted doing", Status: store.StatusDoing},
		{ID: 3, Title: "todo", Status: store.StatusTodo},
	}
	ag := buildAgenda(now, today, lastDay, end, tasks, nil)
	got := inProgressTasks(tasks, ag)
	if len(got) != 1 || got[0].ID != 2 {
		t.Errorf("inProgressTasks = %+v, want only task #2", got)
	}
}

func TestBuildAgendaGroupsAndOverdue(t *testing.T) {
	now := time.Date(2026, 10, 16, 10, 0, 0, 0, time.Local)
	today := startOfLocalDay(now)
	lastDay := today.AddDate(0, 0, 6)
	end := lastDay.AddDate(0, 0, 1).Add(-time.Second)

	at := func(y int, mo time.Month, d, h, mi int) time.Time {
		return time.Date(y, mo, d, h, mi, 0, 0, time.Local)
	}
	overdueAllDay := at(2026, 10, 15, 0, 0)
	todayAllDay := at(2026, 10, 16, 0, 0)
	todayPast := at(2026, 10, 16, 9, 0)
	future := at(2026, 10, 20, 15, 0)
	beyond := at(2026, 10, 30, 0, 0)

	tasks := []store.Task{
		{ID: 1, Title: "overdue all-day", DueAt: &overdueAllDay, AllDay: true},
		{ID: 2, Title: "today all-day", DueAt: &todayAllDay, AllDay: true},
		{ID: 3, Title: "today past", DueAt: &todayPast},
		{ID: 4, Title: "future", DueAt: &future},
		{ID: 5, Title: "beyond window", DueAt: &beyond},
		{ID: 6, Title: "no due"},
	}
	reminders := []store.Reminder{
		{ID: 1, Title: "past reminder", NextAt: at(2026, 10, 16, 8, 0), Enabled: true},
		{ID: 2, Title: "future reminder", NextAt: at(2026, 10, 20, 9, 0), Enabled: true},
	}

	ag := buildAgenda(now, today, lastDay, end, tasks, reminders)

	// Overdue: the all-day task from yesterday, the timed task whose hour has
	// passed today, and the reminder from earlier today.
	if len(ag.overdue) != 3 {
		t.Fatalf("overdue = %d, want 3", len(ag.overdue))
	}
	// Only two days have upcoming items: today and the 20th. The out-of-window
	// and no-due tasks are dropped.
	if len(ag.days) != 2 {
		t.Fatalf("days = %d, want 2", len(ag.days))
	}
	if !ag.days[0].date.Equal(today) || len(ag.days[0].entries) != 1 {
		t.Errorf("today group = %v with %d entries, want 1", ag.days[0].date, len(ag.days[0].entries))
	}
	if got := ag.days[1].date; !got.Equal(at(2026, 10, 20, 0, 0)) {
		t.Errorf("second day = %v, want 2026-10-20", got)
	}
	if len(ag.days[1].entries) != 2 {
		t.Errorf("second day entries = %d, want 2", len(ag.days[1].entries))
	}
}

func TestAgendaToJSONSplitsTasksAndReminders(t *testing.T) {
	now := time.Date(2026, 10, 16, 10, 0, 0, 0, time.Local)
	today := startOfLocalDay(now)
	lastDay := today
	end := lastDay.AddDate(0, 0, 1).Add(-time.Second)

	due := time.Date(2026, 10, 16, 0, 0, 0, 0, time.Local)
	next := time.Date(2026, 10, 16, 11, 0, 0, 0, time.Local)
	ag := buildAgenda(now, today, lastDay, end,
		[]store.Task{{ID: 1, Title: "t", DueAt: &due, AllDay: true}},
		[]store.Reminder{{ID: 2, Title: "r", NextAt: next, Enabled: true}},
	)
	out := overviewToJSON(1, ag, nil)
	if out.RangeDays != 1 {
		t.Errorf("RangeDays = %d, want 1", out.RangeDays)
	}
	if len(out.Schedule) != 1 {
		t.Fatalf("Schedule = %d days, want 1", len(out.Schedule))
	}
	day := out.Schedule[0]
	if len(day.Tasks) != 1 || len(day.Reminders) != 1 {
		t.Errorf("day split = %d tasks / %d reminders, want 1/1", len(day.Tasks), len(day.Reminders))
	}
}
