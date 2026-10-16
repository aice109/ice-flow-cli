package cli

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/aice109/ice-flow-cli/internal/i18n"
	"github.com/aice109/ice-flow-cli/internal/store"
)

// runOverview renders the default dashboard: overdue items, one section per day
// in the requested window, and any in-progress tasks that are not already
// listed above.
func (a *App) runOverview(cmd *cobra.Command, days int) error {
	if err := validateDays(days); err != nil {
		return err
	}
	st, err := a.openStore()
	if err != nil {
		return err
	}
	now := time.Now()
	today := startOfLocalDay(now)
	lastDay := today.AddDate(0, 0, days-1)
	end := lastDay.AddDate(0, 0, 1).Add(-time.Second)

	tasks, err := st.ListTasks(store.TaskFilter{})
	if err != nil {
		return err
	}
	reminders, err := st.ListReminders(true)
	if err != nil {
		return err
	}

	ag := buildAgenda(now, today, lastDay, end, tasks, reminders)
	doing := inProgressTasks(tasks, ag)

	if a.flagJSON {
		return printJSON(overviewToJSON(days, ag, doing))
	}
	a.renderOverview(ag, doing, now, today)
	return nil
}

func validateDays(n int) error {
	if n < 1 {
		return errors.New(i18n.Tf("number of days %d must be at least 1", n))
	}
	if n > 365 {
		return errors.New(i18n.Tf("number of days %d is too large (max 365)", n))
	}
	return nil
}

// inProgressTasks returns the tasks marked "doing" that are not already shown in
// the agenda, so nothing is listed twice.
func inProgressTasks(tasks []store.Task, ag agenda) []store.Task {
	shown := map[int64]bool{}
	for _, e := range ag.overdue {
		if e.task != nil {
			shown[e.task.ID] = true
		}
	}
	for _, d := range ag.days {
		for _, e := range d.entries {
			if e.task != nil {
				shown[e.task.ID] = true
			}
		}
	}
	var out []store.Task
	for _, t := range tasks {
		if t.Status == store.StatusDoing && !shown[t.ID] {
			out = append(out, t)
		}
	}
	return out
}

// agendaEntry is one due item in the agenda: a task or a reminder.
type agendaEntry struct {
	when time.Time
	task *store.Task
	rem  *store.Reminder
}

func (e agendaEntry) allDay() bool { return e.task != nil && e.task.AllDay }

type agendaDay struct {
	date    time.Time
	entries []agendaEntry
}

type agenda struct {
	overdue []agendaEntry
	days    []agendaDay
}

// buildAgenda collects open tasks and enabled reminders that are overdue or
// fall between today and lastDay, grouping future items by their local day.
func buildAgenda(now, today, lastDay, end time.Time, tasks []store.Task, reminders []store.Reminder) agenda {
	var ag agenda
	byDay := map[int64]*agendaDay{}
	var order []int64

	add := func(e agendaEntry) {
		overdue := e.when.Before(now)
		if e.allDay() {
			overdue = startOfLocalDay(e.when).Before(today)
		}
		if overdue {
			ag.overdue = append(ag.overdue, e)
			return
		}
		day := startOfLocalDay(e.when)
		key := day.Unix()
		d, ok := byDay[key]
		if !ok {
			d = &agendaDay{date: day}
			byDay[key] = d
			order = append(order, key)
		}
		d.entries = append(d.entries, e)
	}

	for i := range tasks {
		t := tasks[i]
		if t.Status == store.StatusDone || t.DueAt == nil {
			continue
		}
		due := *t.DueAt
		if t.AllDay {
			// An all-day task is due for the whole of its calendar day.
			if startOfLocalDay(due).After(lastDay) {
				continue
			}
		} else if due.After(end) {
			continue
		}
		add(agendaEntry{when: due, task: &tasks[i]})
	}
	for i := range reminders {
		r := reminders[i]
		if r.NextAt.After(end) {
			continue
		}
		add(agendaEntry{when: r.NextAt, rem: &reminders[i]})
	}

	sort.SliceStable(ag.overdue, func(i, j int) bool { return ag.overdue[i].when.Before(ag.overdue[j].when) })
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	for _, key := range order {
		d := byDay[key]
		sort.SliceStable(d.entries, func(i, j int) bool { return d.entries[i].when.Before(d.entries[j].when) })
		ag.days = append(ag.days, *d)
	}
	return ag
}

func (a *App) renderOverview(ag agenda, doing []store.Task, now, today time.Time) {
	if len(ag.overdue) == 0 && len(ag.days) == 0 && len(doing) == 0 {
		fmt.Println(dim(i18n.T("Nothing on the agenda. Enjoy the quiet.")))
		return
	}
	section := func(title string, entries []agendaEntry, showDate bool) {
		if len(entries) == 0 {
			return
		}
		fmt.Println(bold(title))
		printTable(nil, agendaRows(entries, showDate))
		fmt.Println()
	}
	section(i18n.T("Overdue"), ag.overdue, true)
	for _, d := range ag.days {
		section(agendaDayTitle(d.date, today), d.entries, false)
	}
	if len(doing) > 0 {
		fmt.Println(bold(i18n.T("In progress")))
		printTable(nil, taskRows(doing, now))
		fmt.Println()
	}
}

func agendaDayTitle(day, today time.Time) string {
	date := fmtLocal(day, "Mon 2006-01-02")
	switch {
	case day.Equal(today):
		return i18n.T("Today") + " · " + date
	case day.Equal(today.AddDate(0, 0, 1)):
		return i18n.T("Tomorrow") + " · " + date
	default:
		return date
	}
}

func agendaRows(entries []agendaEntry, showDate bool) [][]string {
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		when := agendaWhen(e, showDate)
		if showDate {
			when = red(when)
		}
		if e.task != nil {
			rows = append(rows, []string{
				statusGlyph(e.task.Status),
				store.PriorityLabel(e.task.Priority),
				"#" + strconv.FormatInt(e.task.ID, 10),
				e.task.Title,
				when,
			})
			continue
		}
		if e.rem.RRule != "" {
			when += " · " + e.rem.RRule
		}
		rows = append(rows, []string{
			"🔔",
			"",
			"#" + strconv.FormatInt(e.rem.ID, 10),
			e.rem.Title,
			when,
		})
	}
	return rows
}

func agendaWhen(e agendaEntry, showDate bool) string {
	if e.allDay() {
		if showDate {
			return fmtLocal(e.when, "2006-01-02") + " (" + i18n.T("all day") + ")"
		}
		return i18n.T("all day")
	}
	if showDate {
		return fmtLocal(e.when, "2006-01-02 15:04")
	}
	return fmtLocal(e.when, "15:04")
}

// --- JSON -----------------------------------------------------------------

type overviewJSON struct {
	RangeDays  int
	Overdue    agendaGroupJSON
	Schedule   []agendaDayJSON
	InProgress []taskJSON
}

type agendaGroupJSON struct {
	Tasks     []taskJSON
	Reminders []reminderJSON
}

type agendaDayJSON struct {
	Date      string
	Tasks     []taskJSON
	Reminders []reminderJSON
}

func overviewToJSON(days int, ag agenda, doing []store.Task) overviewJSON {
	out := overviewJSON{
		RangeDays:  days,
		Overdue:    agendaGroupToJSON(ag.overdue),
		Schedule:   make([]agendaDayJSON, 0, len(ag.days)),
		InProgress: tasksJSON(doing),
	}
	for _, d := range ag.days {
		g := agendaGroupToJSON(d.entries)
		out.Schedule = append(out.Schedule, agendaDayJSON{
			Date:      d.date.Format("2006-01-02"),
			Tasks:     g.Tasks,
			Reminders: g.Reminders,
		})
	}
	return out
}

func agendaGroupToJSON(entries []agendaEntry) agendaGroupJSON {
	g := agendaGroupJSON{Tasks: []taskJSON{}, Reminders: []reminderJSON{}}
	for _, e := range entries {
		if e.task != nil {
			g.Tasks = append(g.Tasks, newTaskJSON(*e.task))
		} else {
			g.Reminders = append(g.Reminders, newReminderJSON(*e.rem))
		}
	}
	return g
}
