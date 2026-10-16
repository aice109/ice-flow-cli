package store

import (
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestBackupSnapshot(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.CreateTask(&Task{Title: "keep"}); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "snapshot.db")
	if err := s.Backup(dest); err != nil {
		t.Fatalf("backup: %v", err)
	}

	snap, err := Open(dest)
	if err != nil {
		t.Fatalf("open snapshot: %v", err)
	}
	defer snap.Close()
	tasks, err := snap.ListTasks(TaskFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Title != "keep" {
		t.Errorf("snapshot tasks = %+v, want one titled keep", tasks)
	}
}

func TestTaskLifecycle(t *testing.T) {
	s := newTestStore(t)

	due := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	id, err := s.CreateTask(&Task{Title: "Write report", Priority: 1, Project: "work", Tags: []string{"a", "b"}, DueAt: &due})
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.GetTask(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Write report" || got.Priority != 1 || got.Status != StatusTodo {
		t.Errorf("unexpected task: %+v", got)
	}
	if len(got.Tags) != 2 || got.DueAt == nil {
		t.Errorf("tags/due not persisted: %+v", got)
	}

	if err := s.SetTaskStatus(id, StatusDone); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetTask(id)
	if got.Status != StatusDone || got.CompletedAt == nil {
		t.Errorf("expected done with completed_at, got %+v", got)
	}

	if err := s.DeleteTask(id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetTask(id); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestTaskFilterAndCascade(t *testing.T) {
	s := newTestStore(t)

	parent, err := s.CreateTask(&Task{Title: "parent", Project: "p", Tags: []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTask(&Task{Title: "child", ParentID: &parent}); err != nil {
		t.Fatal(err)
	}

	roots, err := s.ListTasks(TaskFilter{OnlyRoot: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 1 {
		t.Fatalf("expected 1 root, got %d", len(roots))
	}

	tagged, err := s.ListTasks(TaskFilter{Tag: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tagged) != 1 {
		t.Fatalf("expected 1 tagged task, got %d", len(tagged))
	}

	// Deleting the parent cascades to the child.
	if err := s.DeleteTask(parent); err != nil {
		t.Fatal(err)
	}
	all, _ := s.ListTasks(TaskFilter{})
	if len(all) != 0 {
		t.Errorf("expected cascade delete, %d tasks remain", len(all))
	}
}

func TestDueTasks(t *testing.T) {
	s := newTestStore(t)
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)
	if _, err := s.CreateTask(&Task{Title: "past", DueAt: &past}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTask(&Task{Title: "future", DueAt: &future}); err != nil {
		t.Fatal(err)
	}
	due, err := s.DueTasks(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 || due[0].Title != "past" {
		t.Errorf("unexpected due tasks: %+v", due)
	}
}

func TestReminderScheduleAndFire(t *testing.T) {
	s := newTestStore(t)
	next := time.Now().Add(-time.Minute)
	id, err := s.CreateReminder(&Reminder{Title: "ping", NextAt: next, RRule: "FREQ=DAILY", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}

	due, err := s.DueReminders(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(due) != 1 {
		t.Fatalf("expected 1 due reminder, got %d", len(due))
	}

	later := time.Now().Add(24 * time.Hour)
	if err := s.UpdateReminderNext(id, later); err != nil {
		t.Fatal(err)
	}
	due, _ = s.DueReminders(time.Now())
	if len(due) != 0 {
		t.Errorf("expected no due reminders after advancing, got %d", len(due))
	}

	if err := s.SetReminderEnabled(id, false); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetReminder(id)
	if got.Enabled {
		t.Error("expected reminder disabled")
	}
}

func TestNextDueAt(t *testing.T) {
	s := newTestStore(t)

	if _, ok, err := s.NextDueAt(); err != nil || ok {
		t.Fatalf("NextDueAt on empty store = (%v, %v), want no reminder", ok, err)
	}

	early := time.Now().Add(2 * time.Hour).Truncate(time.Second)
	late := time.Now().Add(5 * time.Hour).Truncate(time.Second)
	earlyID, err := s.CreateReminder(&Reminder{Title: "early", NextAt: early, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateReminder(&Reminder{Title: "late", NextAt: late, Enabled: true}); err != nil {
		t.Fatal(err)
	}

	got, ok, err := s.NextDueAt()
	if err != nil || !ok {
		t.Fatalf("NextDueAt = (%v, %v), want a reminder", ok, err)
	}
	if !got.Equal(early) {
		t.Errorf("NextDueAt = %s, want %s", got, early)
	}

	// Disabling the earliest reminder should surface the next one.
	if err := s.SetReminderEnabled(earlyID, false); err != nil {
		t.Fatal(err)
	}
	got, _, _ = s.NextDueAt()
	if !got.Equal(late) {
		t.Errorf("NextDueAt after disabling = %s, want %s", got, late)
	}
}

func TestTaskAllDayRoundTrip(t *testing.T) {
	s := newTestStore(t)
	due := time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local)
	id, err := s.CreateTask(&Task{Title: "all day", DueAt: &due, AllDay: true})
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.GetTask(id)
	if err != nil {
		t.Fatal(err)
	}
	if !got.AllDay {
		t.Error("AllDay did not round-trip")
	}
	if got.DueAt == nil || !got.DueAt.Equal(due) {
		t.Errorf("DueAt = %v, want %v", got.DueAt, due)
	}

	// Clearing the due date must clear the flag too.
	got.DueAt = nil
	got.AllDay = false
	if err := s.UpdateTask(got); err != nil {
		t.Fatal(err)
	}
	got, _ = s.GetTask(id)
	if got.DueAt != nil || got.AllDay {
		t.Errorf("after clear: DueAt=%v AllDay=%v, want nil/false", got.DueAt, got.AllDay)
	}
}

func TestSnippetUsage(t *testing.T) {
	s := newTestStore(t)
	id, err := s.CreateSnippet(&Snippet{Command: "flutter doctor", Description: "check env", Tags: []string{"flutter"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.RecordSnippetUse(id); err != nil {
		t.Fatal(err)
	}
	got, _ := s.GetSnippet(id)
	if got.UseCount != 1 || got.LastUsedAt == nil {
		t.Errorf("usage not recorded: %+v", got)
	}
	list, _ := s.ListSnippets()
	if len(list) != 1 || list[0].Command != "flutter doctor" {
		t.Errorf("unexpected list: %+v", list)
	}
}

func TestSplitTags(t *testing.T) {
	got := SplitTags(" work, Work , , home ")
	if len(got) != 2 || got[0] != "work" || got[1] != "home" {
		t.Errorf("SplitTags = %v", got)
	}
}
