package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aice109/ice-flow-cli/internal/i18n"
)

// setLocale pins the process locale for a test and restores it afterwards.
func setLocale(t *testing.T, locale string) {
	t.Helper()
	orig := i18n.Locale()
	t.Cleanup(func() { i18n.SetLocale(orig) })
	i18n.SetLocale(locale)
}

// renderHelp builds the command tree (as Execute would) and returns the help
// output for the given args.
func renderHelp(t *testing.T, args ...string) string {
	t.Helper()
	a := &App{}
	root := a.rootCmd()
	localizeTree(root)

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append(args, "--help"))
	if err := root.Execute(); err != nil {
		t.Fatalf("execute %v: %v", args, err)
	}
	return buf.String()
}

func TestHelpIsLocalized(t *testing.T) {
	setLocale(t, i18n.ChineseSimplified)

	top := renderHelp(t)
	for _, want := range []string{"用法：", "计划", "词典", "维护", "帮助", "标志：", "查看命令的更多信息"} {
		if !strings.Contains(top, want) {
			t.Errorf("top-level help missing %q:\n%s", want, top)
		}
	}
	// Every child is grouped, so the fallback section must not appear.
	if strings.Contains(top, "附加命令：") {
		t.Errorf("unexpected ungrouped section:\n%s", top)
	}

	task := renderHelp(t, "task")
	for _, want := range []string{"可用命令：", "全局标志：", "添加任务", "以层级树显示任务"} {
		if !strings.Contains(task, want) {
			t.Errorf("task help missing %q:\n%s", want, task)
		}
	}
}

func TestHelpStaysEnglishByDefault(t *testing.T) {
	setLocale(t, i18n.English)

	out := renderHelp(t)
	for _, want := range []string{"Usage:", "Planning", "Dictionary", "Flags:"} {
		if !strings.Contains(out, want) {
			t.Errorf("English help missing %q:\n%s", want, out)
		}
	}

	task := renderHelp(t, "task")
	for _, want := range []string{"Available Commands:", "Global Flags:", "Add a task"} {
		if !strings.Contains(task, want) {
			t.Errorf("English task help missing %q:\n%s", want, task)
		}
	}
}

func TestLocalizeError(t *testing.T) {
	setLocale(t, i18n.ChineseSimplified)

	cases := map[string]string{
		`unknown command "tsk" for "iflow"`:           `未知命令 "tsk"（属于 "iflow"）`,
		"unknown flag: --bogus":                       "未知标志: --bogus",
		"unknown shorthand flag: 'x' in -x":           "未知短标志: 'x' in -x",
		"accepts 1 arg(s), received 2":                "需要 1 个参数，收到 2 个",
		"requires at least 1 arg(s), only received 0": "至少需要 1 个参数，只收到 0 个",
		"not found": "未找到对应记录",
	}
	for in, want := range cases {
		if got := localizeError(errors.New(in)); got != want {
			t.Errorf("localizeError(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLocalizeErrorKeepsSuggestions(t *testing.T) {
	setLocale(t, i18n.ChineseSimplified)

	in := "unknown command \"tsk\" for \"iflow\"\n\nDid you mean this?\n\ttask\n"
	got := localizeError(errors.New(in))
	if !strings.Contains(got, "Did you mean this?") {
		t.Errorf("lost suggestions: %q", got)
	}
	if !strings.HasPrefix(got, `未知命令 "tsk"（属于 "iflow"）`) {
		t.Errorf("prefix not localized: %q", got)
	}
}

func TestRelativeTimeIsLocalized(t *testing.T) {
	setLocale(t, i18n.ChineseSimplified)
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.Local)

	cases := []struct {
		at   time.Time
		want string
	}{
		{now.Add(13 * time.Hour), "13 小时后"},
		{now.Add(time.Hour), "1 小时后"},
		{now.Add(-90 * time.Minute), "已逾期 1 小时"},
		{now.Add(-30 * time.Second), "逾期不到 1 分钟"},
		{now.Add(30 * time.Second), "不到 1 分钟后"},
		{now.Add(-48 * time.Hour), "已逾期 2 天"},
	}
	for _, c := range cases {
		if got := relTime(c.at, now); got != c.want {
			t.Errorf("relTime(%v) = %q, want %q", c.at, got, c.want)
		}
	}
}
