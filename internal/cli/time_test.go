package cli

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/aice109/ice-flow-cli/internal/store"
)

// withLocal pins time.Local for the duration of a test so timezone behavior is
// deterministic regardless of the machine or CI runner.
func withLocal(t *testing.T, offsetSeconds int) {
	t.Helper()
	orig := time.Local
	time.Local = time.FixedZone("TEST", offsetSeconds)
	t.Cleanup(func() { time.Local = orig })
}

// TestRecurrenceUsesLocalWallClock guards the bug where a reminder entered as
// "every Monday 00:30" (UTC+8) fired on Tuesday, because the recurrence ran on
// the UTC-parsed anchor instead of the local wall clock.
func TestRecurrenceUsesLocalWallClock(t *testing.T) {
	withLocal(t, 8*3600)

	localAnchor := time.Date(2026, 10, 19, 0, 30, 0, 0, time.Local) // Monday 00:30 local
	stored := localAnchor.UTC()                                     // Sunday 16:30 UTC

	next := nextReminderTime(localAnchor, stored, "FREQ=WEEKLY;BYDAY=MO")
	if wd := next.Local().Weekday(); wd != time.Monday {
		t.Fatalf("expected Monday local, got %s (%s local)", wd, next.Local().Format(time.RFC3339))
	}
	want := time.Date(2026, 10, 26, 0, 30, 0, 0, time.Local)
	if !next.Equal(want) {
		t.Errorf("got %v want %v", next, want)
	}
	if h, m := next.Local().Hour(), next.Local().Minute(); h != 0 || m != 30 {
		t.Errorf("wall clock drifted to %02d:%02d", h, m)
	}
}

func TestJSONTimeUsesLocalOffset(t *testing.T) {
	withLocal(t, 8*3600)

	utc := time.Date(2026, 10, 16, 1, 0, 0, 0, time.UTC)
	got := jsonTimePtr(&utc)
	if got == nil || *got != "2026-10-16T09:00:00+08:00" {
		t.Fatalf("jsonTimePtr = %v, want 2026-10-16T09:00:00+08:00", got)
	}
	parsed, err := time.Parse(time.RFC3339, *got)
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.Equal(utc) {
		t.Errorf("round trip lost the instant: %v vs %v", parsed, utc)
	}
}

func TestTaskJSONRendersLocalTime(t *testing.T) {
	withLocal(t, 8*3600)

	due := time.Date(2026, 10, 16, 1, 0, 0, 0, time.UTC)
	created := time.Date(2026, 10, 15, 14, 7, 31, 0, time.UTC)
	b, err := json.Marshal(newTaskJSON(store.Task{ID: 1, Title: "x", DueAt: &due, CreatedAt: created}))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{
		`"DueAt":"2026-10-16T09:00:00+08:00"`,
		`"CreatedAt":"2026-10-15T22:07:31+08:00"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("json %s\nmissing %s", s, want)
		}
	}
}

func TestJSONTimeNilStaysNull(t *testing.T) {
	b, err := json.Marshal(newTaskJSON(store.Task{ID: 1, Title: "x"}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"DueAt":null`) {
		t.Errorf("expected null due, got %s", b)
	}
}
