// Package timeparse turns human-friendly time expressions into time.Time.
//
// Supported forms include absolute dates ("2026-10-20", "2026-10-20 09:00"),
// clocks ("9am", "15:04"), relative durations ("in 30m", "2h"), and day words
// ("today", "tomorrow 9am", "monday 09:00", "next friday").
package timeparse

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aice109/ice-flow-cli/internal/i18n"
)

var durUnitRe = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(w|d|h|m|s)`)

// ParseDuration parses durations with week/day/hour/minute/second units,
// e.g. "1h30m", "2d", "1w", "45m", "90s".
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, errors.New(i18n.T("empty duration"))
	}
	matches := durUnitRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return 0, errors.New(i18n.Tf("invalid duration %q", s))
	}
	// Reject trailing junk like "30mmx" by ensuring the matched span covers
	// everything once whitespace is removed.
	stripped := strings.NewReplacer(" ", "", "\t", "").Replace(s)
	var covered strings.Builder
	var total time.Duration
	for _, m := range matches {
		covered.WriteString(m[0])
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, err
		}
		var unit time.Duration
		switch m[2] {
		case "w":
			unit = 7 * 24 * time.Hour
		case "d":
			unit = 24 * time.Hour
		case "h":
			unit = time.Hour
		case "m":
			unit = time.Minute
		case "s":
			unit = time.Second
		}
		total += time.Duration(v * float64(unit))
	}
	if strings.NewReplacer(" ", "", "\t", "").Replace(covered.String()) != stripped {
		return 0, errors.New(i18n.Tf("invalid duration %q", s))
	}
	return total, nil
}

var layouts = []struct {
	layout string
	allDay bool
}{
	{"2006-01-02 15:04:05", false},
	{"2006-01-02T15:04:05", false},
	{"2006-01-02 15:04", false},
	{"2006-01-02T15:04", false},
	{"2006-01-02", true},
	{"2006/01/02 15:04", false},
	{"2006/01/02", true},
}

// ParseWhen resolves an expression relative to now, returning a specific
// instant. A bare day word ("tomorrow", "monday") keeps the historical
// default time: now when it names today, otherwise 09:00 local. Use
// ParseWhenAllDay when such an expression should be an all-day date instead.
func ParseWhen(expr string, now time.Time) (time.Time, error) {
	t, _, err := parseWhen(expr, now, false)
	return t, err
}

// ParseWhenAllDay resolves an expression relative to now. The bool reports
// whether the expression named a date without a time of day: "2026-10-20",
// "tomorrow" and "monday" are all-day, while "tomorrow 9am", "09:30" and
// "in 3h" are not. For all-day expressions the returned time is the start of
// that day in now's location.
func ParseWhenAllDay(expr string, now time.Time) (time.Time, bool, error) {
	return parseWhen(expr, now, true)
}

// parseWhen implements both entry points. When dayWordAllDay is true a bare
// day word yields the start of that day with allDay set; otherwise it yields
// the historical default instant and allDay is left false.
func parseWhen(expr string, now time.Time, dayWordAllDay bool) (time.Time, bool, error) {
	orig := strings.TrimSpace(expr)
	if orig == "" {
		return time.Time{}, false, errors.New(i18n.T("empty time"))
	}
	lower := strings.ToLower(orig)

	// Relative: "in 30m" or a bare duration "2h".
	if rest, ok := strings.CutPrefix(lower, "in "); ok {
		if d, err := ParseDuration(rest); err == nil {
			return now.Add(d), false, nil
		}
	}
	if d, err := ParseDuration(lower); err == nil {
		return now.Add(d), false, nil
	}

	// Absolute.
	for _, l := range layouts {
		if t, err := time.ParseInLocation(l.layout, orig, now.Location()); err == nil {
			return t, l.allDay, nil
		}
	}

	// Day words / weekdays, optionally followed by a clock.
	tokens := strings.Fields(lower)
	base := time.Time{}
	daySet := false
	rollIfPast := true
	switch {
	case tokens[0] == "today":
		base, daySet = startOfDay(now), true
		tokens = tokens[1:]
	case tokens[0] == "tomorrow" || tokens[0] == "tmr" || tokens[0] == "tom":
		base, daySet = startOfDay(now).AddDate(0, 0, 1), true
		tokens = tokens[1:]
	case tokens[0] == "next" && len(tokens) > 1:
		if wd, ok := weekdayOf(tokens[1]); ok {
			base, daySet = nextWeekday(now, wd, true), true
			tokens = tokens[2:]
			rollIfPast = false
		}
	default:
		if wd, ok := weekdayOf(tokens[0]); ok {
			base, daySet = nextWeekday(now, wd, false), true
			tokens = tokens[1:]
			rollIfPast = false
		}
	}

	rest := strings.Join(tokens, " ")
	if rest == "" {
		if !daySet {
			return time.Time{}, false, errors.New(i18n.Tf("unrecognized time %q", orig))
		}
		if dayWordAllDay {
			// A bare day word ("tomorrow", "monday") is an all-day date, not
			// the moment the command happened to run.
			return base, true, nil
		}
		if sameDay(base, now) {
			return now, false, nil
		}
		return base.Add(9 * time.Hour), false, nil
	}

	h, mi, ok := parseClock(rest)
	if !ok {
		return time.Time{}, false, errors.New(i18n.Tf("unrecognized time %q", orig))
	}
	if !daySet {
		base = startOfDay(now)
	}
	res := base.Add(time.Duration(h)*time.Hour + time.Duration(mi)*time.Minute)
	if (!daySet && !res.After(now)) || (rollIfPast && !res.After(now)) {
		res = res.AddDate(0, 0, 1)
	}
	return res, false, nil
}

var clockRe = regexp.MustCompile(`^(\d{1,2})(?::(\d{2}))?\s*(am|pm)?$`)

func parseClock(s string) (int, int, bool) {
	m := clockRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, 0, false
	}
	h, _ := strconv.Atoi(m[1])
	mi := 0
	if m[2] != "" {
		mi, _ = strconv.Atoi(m[2])
	}
	switch m[3] {
	case "am":
		if h == 12 {
			h = 0
		}
	case "pm":
		if h < 12 {
			h += 12
		}
	}
	if h > 23 || mi > 59 {
		return 0, 0, false
	}
	return h, mi, true
}

var weekdays = map[string]time.Weekday{
	"sun": time.Sunday, "sunday": time.Sunday,
	"mon": time.Monday, "monday": time.Monday,
	"tue": time.Tuesday, "tues": time.Tuesday, "tuesday": time.Tuesday,
	"wed": time.Wednesday, "wednesday": time.Wednesday,
	"thu": time.Thursday, "thur": time.Thursday, "thurs": time.Thursday, "thursday": time.Thursday,
	"fri": time.Friday, "friday": time.Friday,
	"sat": time.Saturday, "saturday": time.Saturday,
}

func weekdayOf(s string) (time.Weekday, bool) {
	wd, ok := weekdays[strings.ToLower(strings.Trim(s, ","))]
	return wd, ok
}

func nextWeekday(now time.Time, wd time.Weekday, forceNext bool) time.Time {
	days := (int(wd) - int(now.Weekday()) + 7) % 7
	if days == 0 && forceNext {
		days = 7
	}
	return startOfDay(now).AddDate(0, 0, days)
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
