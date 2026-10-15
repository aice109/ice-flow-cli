package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestDisplayWidth(t *testing.T) {
	cases := map[string]int{
		"abc":                3,
		"\x1b[31mred\x1b[0m": 3,
		"\x1b[2mID\x1b[0m":   2,
		"中文":                 4,
		"a中b":                4,
		"☐":                  1,
		"":                   0,
		"\x1b[1;33m中\x1b[0m": 2,
	}
	for in, want := range cases {
		if got := displayWidth(in); got != want {
			t.Errorf("displayWidth(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestHumanDurWording(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{10 * time.Second, "<1 min"},
		{time.Minute, "1 min"},
		{5 * time.Minute, "5 mins"},
		{90 * time.Minute, "1 hr"},
		{13 * time.Hour, "13 hrs"},
		{24 * time.Hour, "1 day"},
		{72 * time.Hour, "3 days"},
	}
	for _, c := range cases {
		if got := humanDur(c.d); got != c.want {
			t.Errorf("humanDur(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestRelTime(t *testing.T) {
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.Local)
	if got := relTime(now.Add(13*time.Hour), now); got != "in 13 hrs" {
		t.Errorf("future: got %q", got)
	}
	if got := relTime(now.Add(-time.Minute), now); got != "1 min overdue" {
		t.Errorf("past: got %q", got)
	}
}

// TestTableAlignmentIgnoresColor verifies that colored output occupies exactly
// the same visible layout as uncolored output.
func TestTableAlignmentIgnoresColor(t *testing.T) {
	headers := func() []string {
		return []string{dim("ID"), "", dim("P"), dim("Due"), dim("Project"), dim("Title"), dim("Tags")}
	}
	rows := func() [][]string {
		return [][]string{
			{"1", statusGlyph("todo"), "P2", red("2026-10-15 09:58 (1 min overdue)"), dim("-"), "sort my task", ""},
			{"22", statusGlyph("doing"), "P0", yellow("2026-10-15 23:00 (in 13 hrs)"), cyan("reports"), "fix 表格对齐", dim("#work #ui")},
		}
	}

	colorEnabled = true
	var colored bytes.Buffer
	writeTable(&colored, headers(), rows())

	colorEnabled = false
	var plain bytes.Buffer
	writeTable(&plain, headers(), rows())

	got := stripANSI(colored.String())
	if got != plain.String() {
		t.Errorf("colored layout differs from plain layout\n--- colored (stripped) ---\n%s\n--- plain ---\n%s", got, plain.String())
	}
}

func TestTableAlignmentColumnsMatch(t *testing.T) {
	colorEnabled = false
	defer func() { colorEnabled = true }()

	var buf bytes.Buffer
	writeTable(&buf, []string{"ID", "Title"}, [][]string{
		{"1", "short"},
		{"22", "much longer title"},
	})
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), lines)
	}
	// The second column must begin at the same offset on every line.
	want := strings.Index(lines[0], "Title")
	for _, ln := range lines[1:] {
		fields := strings.Fields(ln)
		if len(fields) < 2 {
			t.Fatalf("unexpected line %q", ln)
		}
		second := strings.Index(ln, fields[1])
		if second != want {
			t.Errorf("column misaligned on %q: second column at %d, want %d", ln, second, want)
		}
	}
}
