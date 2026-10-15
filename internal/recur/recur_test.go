package recur

import (
	"testing"
	"time"
)

func TestParseEvery(t *testing.T) {
	cases := map[string]string{
		"day":         "FREQ=DAILY",
		"daily":       "FREQ=DAILY",
		"weekday":     "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
		"weekdays":    "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
		"every 2d":    "FREQ=DAILY;INTERVAL=2",
		"3w":          "FREQ=WEEKLY;INTERVAL=3",
		"6h":          "FREQ=HOURLY;INTERVAL=6",
		"mon,wed,fri": "FREQ=WEEKLY;BYDAY=MO,WE,FR",
		"monthly":     "FREQ=MONTHLY",
	}
	for in, want := range cases {
		got, err := ParseEvery(in)
		if err != nil {
			t.Errorf("ParseEvery(%q) error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseEvery(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := ParseEvery("fortnight"); err == nil {
		t.Error("expected error for unknown recurrence")
	}
}

func TestNextDaily(t *testing.T) {
	anchor := time.Date(2026, 10, 15, 9, 0, 0, 0, time.Local)
	after := time.Date(2026, 10, 15, 10, 0, 0, 0, time.Local)
	got := Next(after, anchor, "FREQ=DAILY")
	want := time.Date(2026, 10, 16, 9, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestNextDailyInterval(t *testing.T) {
	anchor := time.Date(2026, 10, 15, 9, 0, 0, 0, time.Local)
	after := time.Date(2026, 10, 16, 10, 0, 0, 0, time.Local)
	got := Next(after, anchor, "FREQ=DAILY;INTERVAL=2")
	want := time.Date(2026, 10, 17, 9, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestNextWeeklyByDay(t *testing.T) {
	// Anchor is a Friday; business-days rule should jump to Monday.
	anchor := time.Date(2026, 10, 16, 9, 30, 0, 0, time.Local)
	after := time.Date(2026, 10, 16, 9, 30, 0, 0, time.Local)
	got := Next(after, anchor, "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR")
	want := time.Date(2026, 10, 19, 9, 30, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestNextMonthlyClamps(t *testing.T) {
	// Jan 31 -> Feb 28 (non-leap 2026).
	anchor := time.Date(2026, 1, 31, 8, 0, 0, 0, time.Local)
	after := time.Date(2026, 1, 31, 9, 0, 0, 0, time.Local)
	got := Next(after, anchor, "FREQ=MONTHLY")
	want := time.Date(2026, 2, 28, 8, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestNextEmpty(t *testing.T) {
	if got := Next(time.Now(), time.Now(), ""); !got.IsZero() {
		t.Errorf("expected zero time, got %v", got)
	}
}
