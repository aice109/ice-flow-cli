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
