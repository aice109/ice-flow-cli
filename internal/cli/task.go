package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/aice109/ice-flow-cli/internal/store"
	"github.com/aice109/ice-flow-cli/internal/timeparse"
)

func (a *App) taskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Plan and track tasks",
		Long:  "Create, list and complete tasks. Tasks may be nested via --parent and grouped with --project/--tag.",
	}
	cmd.AddCommand(
		a.taskAddCmd(),
		a.taskListCmd(),
		a.taskTodayCmd(),
		a.taskTreeCmd(),
		a.taskStatusCmd("start", "Mark a task as in progress", store.StatusDoing),
		a.taskStatusCmd("done", "Mark a task as done", store.StatusDone),
		a.taskStatusCmd("reopen", "Reopen a task", store.StatusTodo),
		a.taskEditCmd(),
		a.taskRmCmd(),
	)
	return cmd
}

func (a *App) taskAddCmd() *cobra.Command {
	var (
		due     string
		prio    string
		tags    []string
		project string
		notes   string
		parent  int64
	)
	cmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Add a task",
		Example: `  iflow task add "Write weekly report" --due 2026-10-20 --prio high --tag work
  iflow task add "Design API" --project platform --parent 12`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := a.openStore()
			if err != nil {
				return err
			}
			p := a.Cfg.DefaultPriority
			if prio != "" {
				if p, err = parsePriority(prio); err != nil {
					return err
				}
			}
			t := &store.Task{
				Title:    strings.Join(args, " "),
				Notes:    notes,
				Status:   store.StatusTodo,
				Priority: p,
				Project:  project,
				Tags:     tags,
			}
			if due != "" {
				d, err := parseDue(due)
				if err != nil {
					return err
				}
				t.DueAt = &d
			}
			if parent != 0 {
				if _, err := st.GetTask(parent); err != nil {
					return fmt.Errorf("parent task %d: %w", parent, err)
				}
				t.ParentID = &parent
			}
			id, err := st.CreateTask(t)
			if err != nil {
				return err
			}
			fmt.Printf("added task %s\n", bold(fmt.Sprintf("#%d", id)))
			return nil
		},
	}
	cmd.Flags().StringVar(&due, "due", "", "due date/time (e.g. 2026-10-20, tomorrow 9am, in 3h)")
	cmd.Flags().StringVar(&prio, "prio", "", "priority: p0..p3, 0..3, or high/med/low")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "tags (repeatable or comma-separated)")
	cmd.Flags().StringVar(&project, "project", "", "project name")
	cmd.Flags().StringVar(&notes, "notes", "", "free-form notes")
	cmd.Flags().Int64Var(&parent, "parent", 0, "parent task id")
	return cmd
}

func (a *App) taskListCmd() *cobra.Command {
	var (
		status  string
		tag     string
		project string
		all     bool
	)
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List tasks (open by default; --all includes done)",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := a.openStore()
			if err != nil {
				return err
			}
			tasks, err := st.ListTasks(store.TaskFilter{Status: status, Tag: tag, Project: project})
			if err != nil {
				return err
			}
			if !all && status == "" {
				tasks = filterOutDone(tasks)
			}
			return a.outputTasks(tasks)
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "filter by status: todo|doing|done")
	cmd.Flags().StringVar(&tag, "tag", "", "filter by tag")
	cmd.Flags().StringVar(&project, "project", "", "filter by project")
	cmd.Flags().BoolVar(&all, "all", false, "include completed tasks")
	return cmd
}

func (a *App) taskTodayCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "today",
		Short: "Show overdue, due-today and in-progress tasks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runToday(cmd)
		},
	}
}

func (a *App) taskTreeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tree",
		Short: "Show tasks as a hierarchy",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := a.openStore()
			if err != nil {
				return err
			}
			tasks, err := st.ListTasks(store.TaskFilter{})
			if err != nil {
				return err
			}
			if a.flagJSON {
				return printJSON(tasks)
			}
			renderTree(tasks)
			return nil
		},
	}
}

func (a *App) taskStatusCmd(name, short, status string) *cobra.Command {
	return &cobra.Command{
		Use:   name + " <id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id %q", args[0])
			}
			st, err := a.openStore()
			if err != nil {
				return err
			}
			if err := st.SetTaskStatus(id, status); err != nil {
				return err
			}
			fmt.Printf("task #%d -> %s\n", id, status)
			return nil
		},
	}
}

func (a *App) taskEditCmd() *cobra.Command {
	var (
		title   string
		notes   string
		status  string
		prio    string
		project string
		due     string
		tags    []string
		addTags []string
		rmTags  []string
		clearD  bool
	)
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit a task's fields",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id %q", args[0])
			}
			st, err := a.openStore()
			if err != nil {
				return err
			}
			t, err := st.GetTask(id)
			if err != nil {
				return err
			}
			if title != "" {
				t.Title = title
			}
			if notes != "" {
				t.Notes = notes
			}
			if status != "" {
				t.Status = status
				if status == store.StatusDone {
					now := time.Now()
					t.CompletedAt = &now
				}
			}
			if prio != "" {
				if t.Priority, err = parsePriority(prio); err != nil {
					return err
				}
			}
			if project != "" {
				t.Project = project
			}
			if due != "" {
				d, err := parseDue(due)
				if err != nil {
					return err
				}
				t.DueAt = &d
			}
			if clearD {
				t.DueAt = nil
			}
			if len(tags) > 0 {
				t.Tags = tags
			}
			t.Tags = appendUnique(t.Tags, addTags...)
			t.Tags = removeAll(t.Tags, rmTags)
			if err := st.UpdateTask(t); err != nil {
				return err
			}
			fmt.Printf("updated task #%d\n", id)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "new title")
	cmd.Flags().StringVar(&notes, "notes", "", "new notes")
	cmd.Flags().StringVar(&status, "status", "", "new status: todo|doing|done")
	cmd.Flags().StringVar(&prio, "prio", "", "new priority")
	cmd.Flags().StringVar(&project, "project", "", "new project")
	cmd.Flags().StringVar(&due, "due", "", "new due date/time")
	cmd.Flags().BoolVar(&clearD, "clear-due", false, "remove the due date")
	cmd.Flags().StringSliceVar(&tags, "tag", nil, "replace tags")
	cmd.Flags().StringSliceVar(&addTags, "add-tag", nil, "add tags")
	cmd.Flags().StringSliceVar(&rmTags, "rm-tag", nil, "remove tags")
	return cmd
}

func (a *App) taskRmCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "rm <id>",
		Aliases: []string{"delete", "remove"},
		Short:   "Delete a task (and its subtasks)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid id %q", args[0])
			}
			st, err := a.openStore()
			if err != nil {
				return err
			}
			if !yes && !confirm(fmt.Sprintf("Delete task #%d and its subtasks?", id)) {
				return fmt.Errorf("aborted (use --yes to skip confirmation)")
			}
			if err := st.DeleteTask(id); err != nil {
				return err
			}
			fmt.Printf("deleted task #%d\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation")
	return cmd
}

// --- shared rendering -----------------------------------------------------

func (a *App) runToday(cmd *cobra.Command) error {
	st, err := a.openStore()
	if err != nil {
		return err
	}
	now := time.Now()
	tasks, err := st.ListTasks(store.TaskFilter{})
	if err != nil {
		return err
	}
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	var overdue, today, doing []store.Task
	for _, t := range tasks {
		if t.Status == store.StatusDone {
			continue
		}
		switch {
		case t.Status == store.StatusDoing:
			doing = append(doing, t)
		}
		if t.DueAt != nil {
			if t.DueAt.Before(now) {
				overdue = append(overdue, t)
			} else if !t.DueAt.After(endOfDay) {
				today = append(today, t)
			}
		}
	}
	if a.flagJSON {
		return printJSON(map[string][]store.Task{"overdue": overdue, "today": today, "doing": doing})
	}
	if len(overdue) == 0 && len(today) == 0 && len(doing) == 0 {
		fmt.Println(dim("Nothing on the agenda. Enjoy the quiet."))
		return nil
	}
	section := func(title string, ts []store.Task) {
		if len(ts) == 0 {
			return
		}
		fmt.Println(bold(title))
		rows := taskRows(ts, now)
		printTable(nil, rows)
		fmt.Println()
	}
	section("Overdue", overdue)
	section("Due today", today)
	section("In progress", doing)
	return nil
}

func (a *App) outputTasks(tasks []store.Task) error {
	if a.flagJSON {
		return printJSON(tasks)
	}
	if len(tasks) == 0 {
		fmt.Println(dim("No tasks."))
		return nil
	}
	printTable([]string{dim("ID"), "", dim("P"), dim("Due"), dim("Project"), dim("Title"), dim("Tags")}, taskRows(tasks, time.Now()))
	return nil
}

func taskRows(tasks []store.Task, now time.Time) [][]string {
	rows := make([][]string, 0, len(tasks))
	for _, t := range tasks {
		title := t.Title
		if t.Status == store.StatusDone {
			title = dim(title)
		}
		rows = append(rows, []string{
			strconv.FormatInt(t.ID, 10),
			statusGlyph(t.Status),
			store.PriorityLabel(t.Priority),
			fmtDue(t.DueAt, now),
			projectLabel(t.Project),
			title,
			tagsLabel(t.Tags),
		})
	}
	return rows
}

func renderTree(tasks []store.Task) {
	children := map[int64][]store.Task{}
	byID := map[int64]store.Task{}
	for _, t := range tasks {
		byID[t.ID] = t
	}
	var roots []store.Task
	for _, t := range tasks {
		if t.ParentID != nil {
			if _, ok := byID[*t.ParentID]; ok {
				children[*t.ParentID] = append(children[*t.ParentID], t)
				continue
			}
		}
		roots = append(roots, t)
	}
	now := time.Now()
	var walk func(t store.Task, depth int)
	walk = func(t store.Task, depth int) {
		indent := strings.Repeat("  ", depth)
		line := fmt.Sprintf("%s%s %s #%s %s", indent, statusGlyph(t.Status), store.PriorityLabel(t.Priority), strconv.FormatInt(t.ID, 10), t.Title)
		if t.DueAt != nil {
			line += " " + dim("("+fmtDue(t.DueAt, now)+")")
		}
		fmt.Println(line)
		for _, c := range children[t.ID] {
			walk(c, depth+1)
		}
	}
	for _, r := range roots {
		walk(r, 0)
	}
}

func filterOutDone(tasks []store.Task) []store.Task {
	out := tasks[:0]
	for _, t := range tasks {
		if t.Status != store.StatusDone {
			out = append(out, t)
		}
	}
	return out
}

// --- parsing helpers ------------------------------------------------------

func parsePriority(s string) (int, error) {
	v := strings.ToLower(strings.TrimSpace(s))
	v = strings.TrimPrefix(v, "p")
	switch v {
	case "0", "urgent", "critical":
		return 0, nil
	case "1", "high":
		return 1, nil
	case "2", "medium", "med", "normal":
		return 2, nil
	case "3", "low":
		return 3, nil
	}
	if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 3 {
		return n, nil
	}
	return 0, fmt.Errorf("invalid priority %q (use p0..p3 or high/med/low)", s)
}

func parseDue(s string) (time.Time, error) {
	return timeparse.ParseWhen(s, time.Now())
}

func projectLabel(p string) string {
	if p == "" {
		return dim("-")
	}
	return cyan(p)
}

func tagsLabel(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	return dim("#" + strings.Join(tags, " #"))
}

func appendUnique(dst []string, add ...string) []string {
	seen := map[string]bool{}
	for _, t := range dst {
		seen[strings.ToLower(t)] = true
	}
	for _, t := range add {
		t = strings.TrimSpace(t)
		if t == "" || seen[strings.ToLower(t)] {
			continue
		}
		seen[strings.ToLower(t)] = true
		dst = append(dst, t)
	}
	return dst
}

func removeAll(dst []string, rm []string) []string {
	if len(rm) == 0 {
		return dst
	}
	drop := map[string]bool{}
	for _, t := range rm {
		drop[strings.ToLower(strings.TrimSpace(t))] = true
	}
	out := dst[:0]
	for _, t := range dst {
		if !drop[strings.ToLower(t)] {
			out = append(out, t)
		}
	}
	return out
}
