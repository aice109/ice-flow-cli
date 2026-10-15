// Package recur parses a small subset of iCalendar RRULEs and computes the
// next occurrence of a recurring reminder.
//
// Supported forms:
//
//	FREQ=HOURLY
//	FREQ=DAILY;INTERVAL=2
//	FREQ=WEEKLY
//	FREQ=WEEKLY;BYDAY=MO,WE,FR
//	FREQ=MONTHLY
package recur

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aice109/ice-flow-cli/internal/i18n"
)

var intervalRe = regexp.MustCompile(`^(\d+)\s*(h|d|w|m)$`)

// ParseEvery maps a friendly spec to an RRULE string. Accepts "day"/"daily",
// "hour"/"hourly", "week"/"weekly", "month"/"monthly", "weekday"/"weekdays",
// weekday lists such as "mon,wed,fri", and intervals such as "2d", "3w", "6h".
func ParseEvery(spec string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(spec))
	s = strings.TrimPrefix(s, "every ")
	s = strings.ReplaceAll(s, " ", "")
	switch s {
	case "", "day", "daily":
		return "FREQ=DAILY", nil
	case "hour", "hourly":
		return "FREQ=HOURLY", nil
	case "week", "weekly":
		return "FREQ=WEEKLY", nil
	case "month", "monthly":
		return "FREQ=MONTHLY", nil
	case "weekday", "weekdays", "business", "businessday", "businessdays":
		return "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR", nil
	}

	if m := intervalRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		var freq string
		switch m[2] {
		case "h":
			freq = "HOURLY"
		case "d":
			freq = "DAILY"
		case "w":
			freq = "WEEKLY"
		case "m":
			freq = "MONTHLY"
		}
		if n <= 1 {
			return "FREQ=" + freq, nil
		}
		return fmt.Sprintf("FREQ=%s;INTERVAL=%d", freq, n), nil
	}

	// Weekday list, e.g. "mon,wed,fri".
	var codes []string
	for _, part := range strings.Split(s, ",") {
		code, ok := weekdayCode(part)
		if !ok {
			return "", errors.New(i18n.Tf("unrecognized recurrence %q", spec))
		}
		codes = append(codes, code)
	}
	if len(codes) > 0 {
		return "FREQ=WEEKLY;BYDAY=" + strings.Join(codes, ","), nil
	}
	return "", errors.New(i18n.Tf("unrecognized recurrence %q", spec))
}

func weekdayCode(s string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "su", "sun", "sunday":
		return "SU", true
	case "mo", "mon", "monday":
		return "MO", true
	case "tu", "tue", "tues", "tuesday":
		return "TU", true
	case "we", "wed", "wednesday":
		return "WE", true
	case "th", "thu", "thur", "thurs", "thursday":
		return "TH", true
	case "fr", "fri", "friday":
		return "FR", true
	case "sa", "sat", "saturday":
		return "SA", true
	}
	return "", false
}

type rule struct {
	freq     string
	interval int
	days     map[time.Weekday]bool
}

func parse(rrule string) rule {
	r := rule{interval: 1, days: map[time.Weekday]bool{}}
	for _, part := range strings.Split(rrule, ";") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(kv[0])) {
		case "FREQ":
			r.freq = strings.ToUpper(strings.TrimSpace(kv[1]))
		case "INTERVAL":
			if n, err := strconv.Atoi(strings.TrimSpace(kv[1])); err == nil && n > 0 {
				r.interval = n
			}
		case "BYDAY":
			for _, d := range strings.Split(kv[1], ",") {
				if wd, ok := dayFromCode(strings.TrimSpace(d)); ok {
					r.days[wd] = true
				}
			}
		}
	}
	return r
}

func dayFromCode(code string) (time.Weekday, bool) {
	switch strings.ToUpper(code) {
	case "SU":
		return time.Sunday, true
	case "MO":
		return time.Monday, true
	case "TU":
		return time.Tuesday, true
	case "WE":
		return time.Wednesday, true
	case "TH":
		return time.Thursday, true
	case "FR":
		return time.Friday, true
	case "SA":
		return time.Saturday, true
	}
	return 0, false
}

// Next returns the first occurrence strictly after `after`, given the anchor
// (the original fire time, which carries the desired clock). It returns the
// zero time when rrule is empty or unrecognized.
func Next(after, anchor time.Time, rrule string) time.Time {
	if strings.TrimSpace(rrule) == "" {
		return time.Time{}
	}
	r := parse(rrule)
	loc := anchor.Location()

	switch r.freq {
	case "HOURLY":
		t := anchor
		for i := 0; i < 100000 && !t.After(after); i++ {
			t = t.Add(time.Duration(r.interval) * time.Hour)
		}
		return t
	case "DAILY":
		t := anchor
		for i := 0; i < 100000 && !t.After(after); i++ {
			t = t.AddDate(0, 0, r.interval)
		}
		return t
	case "WEEKLY":
		days := r.days
		if len(days) == 0 {
			days = map[time.Weekday]bool{anchor.Weekday(): true}
		}
		weekStart := startOfWeek(anchor)
		for w := 0; w < 5000; w++ {
			ws := weekStart.AddDate(0, 0, 7*w*r.interval)
			for d := 0; d < 7; d++ {
				day := ws.AddDate(0, 0, d)
				if !days[day.Weekday()] {
					continue
				}
				cand := time.Date(day.Year(), day.Month(), day.Day(),
					anchor.Hour(), anchor.Minute(), anchor.Second(), 0, loc)
				if cand.After(after) {
					return cand
				}
			}
		}
		return time.Time{}
	case "MONTHLY":
		day := anchor.Day()
		monthStart := time.Date(anchor.Year(), anchor.Month(), 1, 0, 0, 0, 0, loc)
		for i := 0; i < 5000; i++ {
			m := monthStart.AddDate(0, i*r.interval, 0)
			cand := time.Date(m.Year(), m.Month(), min(day, daysIn(m.Year(), m.Month())),
				anchor.Hour(), anchor.Minute(), anchor.Second(), 0, loc)
			if cand.After(after) {
				return cand
			}
		}
		return time.Time{}
	default:
		return time.Time{}
	}
}

func startOfWeek(t time.Time) time.Time {
	y, m, d := t.Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	offset := (int(day.Weekday()) + 6) % 7 // Monday = 0
	return day.AddDate(0, 0, -offset)
}

func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
