package timeparse

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	cases := map[string]time.Duration{
		"30m":    30 * time.Minute,
		"2h":     2 * time.Hour,
		"1d":     24 * time.Hour,
		"1w":     7 * 24 * time.Hour,
		"1h30m":  90 * time.Minute,
		"45s":    45 * time.Second,
		" 10 m ": 10 * time.Minute,
	}
	for in, want := range cases {
		got, err := ParseDuration(in)
		if err != nil {
			t.Errorf("ParseDuration(%q) error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseDuration(%q) = %v, want %v", in, got, want)
		}
	}
	if _, err := ParseDuration("nonsense"); err == nil {
		t.Error("expected error for nonsense duration")
	}
	if _, err := ParseDuration("30mmx"); err == nil {
		t.Error("expected error for trailing junk")
	}
}

func TestParseWhenRelative(t *testing.T) {
	now := time.Date(2026, 10, 15, 21, 0, 0, 0, time.Local) // Thursday
	got, err := ParseWhen("in 30m", now)
	if err != nil {
		t.Fatal(err)
	}
	if want := now.Add(30 * time.Minute); !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestParseWhenTomorrow(t *testing.T) {
	now := time.Date(2026, 10, 15, 21, 0, 0, 0, time.Local)
	got, err := ParseWhen("tomorrow 9am", now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 16, 9, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestParseWhenAbsolute(t *testing.T) {
	now := time.Date(2026, 10, 15, 21, 0, 0, 0, time.Local)
	got, err := ParseWhen("2026-10-20", now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestParseWhenWeekday(t *testing.T) {
	now := time.Date(2026, 10, 15, 21, 0, 0, 0, time.Local) // Thursday
	got, err := ParseWhen("monday 10:00", now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 19, 10, 0, 0, 0, time.Local) // next Monday
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestParseWhenClockRollsOver(t *testing.T) {
	now := time.Date(2026, 10, 15, 21, 0, 0, 0, time.Local)
	got, err := ParseWhen("08:00", now)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 10, 16, 8, 0, 0, 0, time.Local) // already past -> tomorrow
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestParseWhenBareDayDefaultsTime(t *testing.T) {
	now := time.Date(2026, 10, 15, 21, 0, 0, 0, time.Local)
	got, err := ParseWhen("tomorrow", now)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 10, 16, 9, 0, 0, 0, time.Local); !got.Equal(want) {
		t.Errorf("ParseWhen(tomorrow) = %v, want %v", got, want)
	}
	got, err = ParseWhen("today", now)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(now) {
		t.Errorf("ParseWhen(today) = %v, want %v", got, now)
	}
}

func TestParseWhenAllDay(t *testing.T) {
	now := time.Date(2026, 10, 15, 21, 0, 0, 0, time.Local) // Thursday
	cases := []struct {
		expr   string
		allDay bool
		want   time.Time
	}{
		{"today", true, time.Date(2026, 10, 15, 0, 0, 0, 0, time.Local)},
		{"tomorrow", true, time.Date(2026, 10, 16, 0, 0, 0, 0, time.Local)},
		{"monday", true, time.Date(2026, 10, 19, 0, 0, 0, 0, time.Local)},
		{"2026-10-20", true, time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local)},
		{"2026/10/20", true, time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local)},
		{"tomorrow 9am", false, time.Date(2026, 10, 16, 9, 0, 0, 0, time.Local)},
		{"2026-10-20 15:00", false, time.Date(2026, 10, 20, 15, 0, 0, 0, time.Local)},
		{"09:30", false, time.Date(2026, 10, 16, 9, 30, 0, 0, time.Local)},
		{"in 2h", false, now.Add(2 * time.Hour)},
	}
	for _, c := range cases {
		got, allDay, err := ParseWhenAllDay(c.expr, now)
		if err != nil {
			t.Errorf("ParseWhenAllDay(%q) error: %v", c.expr, err)
			continue
		}
		if allDay != c.allDay {
			t.Errorf("ParseWhenAllDay(%q) allDay = %v, want %v", c.expr, allDay, c.allDay)
		}
		if !got.Equal(c.want) {
			t.Errorf("ParseWhenAllDay(%q) = %v, want %v", c.expr, got, c.want)
		}
	}
}
