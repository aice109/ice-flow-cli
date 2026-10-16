package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/aice109/ice-flow-cli/internal/i18n"
	"github.com/aice109/ice-flow-cli/internal/notify"
	"github.com/aice109/ice-flow-cli/internal/recur"
	"github.com/aice109/ice-flow-cli/internal/store"
	"github.com/aice109/ice-flow-cli/internal/timeparse"
)

func (a *App) remindCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remind",
		Short: "Schedule notifications",
		Long:  i18n.T("Schedule one-shot or recurring reminders. Install the background job with `iceflow service install` so they fire even when iceflow is not running."),
	}
	cmd.AddCommand(
		a.remindAddCmd(),
		a.remindListCmd(),
		a.remindRmCmd(),
		a.remindSnoozeCmd(),
		a.remindToggleCmd("enable", true),
		a.remindToggleCmd("disable", false),
		a.remindDaemonCmd(),
	)
	return cmd
}

func (a *App) remindAddCmd() *cobra.Command {
	var at, in, every, notes string
	cmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Add a reminder",
		Example: `  iceflow remind add "Standup" --every weekday --at 09:30
  iceflow remind add "Call mom" --at "tomorrow 19:00"
  iceflow remind add "Stretch" --in 45m
  iceflow remind add "Water plants" --every 2d`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := a.openStore()
			if err != nil {
				return err
			}
			now := time.Now()
			r := &store.Reminder{Title: joinArgs(args), Notes: notes, Enabled: true}

			switch {
			case every != "":
				rrule, err := recur.ParseEvery(every)
				if err != nil {
					return err
				}
				r.RRule = rrule
				anchor := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
				if at != "" {
					if anchor, err = timeparse.ParseWhen(at, now); err != nil {
						return err
					}
				} else if in != "" {
					d, err := timeparse.ParseDuration(in)
					if err != nil {
						return err
					}
					anchor = now.Add(d)
				}
				// Ask the recurrence for the first occurrence at or after now.
				// Parsing a bare clock ("08:00") can roll the anchor onto a day
				// the rule does not allow, so let the rule decide.
				r.NextAt = nextReminderTime(now.Add(-time.Nanosecond), anchor, rrule)
			case in != "":
				d, err := timeparse.ParseDuration(in)
				if err != nil {
					return err
				}
				r.NextAt = now.Add(d)
			case at != "":
				if r.NextAt, err = timeparse.ParseWhen(at, now); err != nil {
					return err
				}
			default:
				return errors.New(i18n.T("provide --at, --in or --every"))
			}

			id, err := st.CreateReminder(r)
			if err != nil {
				return err
			}
			repeat := i18n.T("once")
			if r.RRule != "" {
				repeat = r.RRule
			}
			fmt.Println(i18n.Tf("added reminder %s (next %s, %s)",
				bold(fmt.Sprintf("#%d", id)), fmtLocal(r.NextAt, "2006-01-02 15:04"), repeat))
			return nil
		},
	}
	cmd.Flags().StringVar(&at, "at", "", "absolute time (e.g. 2026-10-16 09:00, tomorrow 9am, 09:30)")
	cmd.Flags().StringVar(&in, "in", "", "relative delay (e.g. 30m, 2h, 1d)")
	cmd.Flags().StringVar(&every, "every", "", "recurrence (day|weekday|week|month|hour|mon,wed,fri)")
	cmd.Flags().StringVar(&notes, "notes", "", "extra text shown in the notification")
	return cmd
}

func (a *App) remindListCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List reminders",
		Example: `  iceflow remind list
  iceflow remind list --all`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := a.openStore()
			if err != nil {
				return err
			}
			rs, err := st.ListReminders(!all)
			if err != nil {
				return err
			}
			if a.flagJSON {
				return printJSON(remindersJSON(rs))
			}
			if len(rs) == 0 {
				fmt.Println(dim(i18n.T("No reminders.")))
				return nil
			}
			now := time.Now()
			rows := make([][]string, 0, len(rs))
			for _, r := range rs {
				state := green(i18n.T("on"))
				if !r.Enabled {
					state = dim(i18n.T("off"))
				}
				repeat := i18n.T("once")
				if r.RRule != "" {
					repeat = r.RRule
				}
				next := dim(i18n.T("paused"))
				if r.Enabled {
					next = fmt.Sprintf("%s (%s)", fmtLocal(r.NextAt, "2006-01-02 15:04"), relTime(r.NextAt, now))
				}
				rows = append(rows, []string{
					strconv.FormatInt(r.ID, 10), state, next, repeat, r.Title,
				})
			}
			printTable([]string{dim(i18n.T("ID")), dim(i18n.T("State")), dim(i18n.T("Next")), dim(i18n.T("Repeat")), dim(i18n.T("Title"))}, rows)
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "include disabled reminders")
	return cmd
}

func (a *App) remindRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "rm <id>",
		Aliases: []string{"delete", "remove"},
		Short:   "Delete a reminder",
		Example: "  iceflow remind rm 3",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			st, err := a.openStore()
			if err != nil {
				return err
			}
			if err := st.DeleteReminder(id); err != nil {
				return err
			}
			fmt.Println(i18n.Tf("deleted reminder #%d", id))
			return nil
		},
	}
}

func (a *App) remindSnoozeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "snooze <id> <duration>",
		Short: "Push a reminder forward (e.g. snooze 3 10m)",
		Example: `  iceflow remind snooze 3 10m
  iceflow remind snooze 3 2h`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			d, err := timeparse.ParseDuration(args[1])
			if err != nil {
				return err
			}
			st, err := a.openStore()
			if err != nil {
				return err
			}
			next := time.Now().Add(d)
			if err := st.SnoozeReminder(id, next); err != nil {
				return err
			}
			fmt.Println(i18n.Tf("snoozed #%d until %s", id, fmtLocal(next, "2006-01-02 15:04")))
			return nil
		},
	}
}

func (a *App) remindToggleCmd(name string, enabled bool) *cobra.Command {
	return &cobra.Command{
		Use:     name + " <id>",
		Short:   name + " a reminder",
		Example: "  iceflow remind " + name + " 3",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return err
			}
			st, err := a.openStore()
			if err != nil {
				return err
			}
			if err := st.SetReminderEnabled(id, enabled); err != nil {
				return err
			}
			fmt.Println(i18n.Tf("reminder #%d %sd", id, i18n.T(name)))
			return nil
		},
	}
}

func (a *App) remindDaemonCmd() *cobra.Command {
	var (
		tick     bool
		interval time.Duration
	)
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Fire due reminders (running loop, or --tick for a single pass)",
		Long: i18n.T("Runs the reminder scheduler.") + "\n\n" +
			i18n.T("Without --tick it stays resident and wakes exactly when the next reminder is due; this is what the installed background service runs. With --tick it performs a single pass and exits, for cron-style schedulers."),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := a.openStore()
			if err != nil {
				return err
			}
			fire := func() {
				if _, err := a.tick(st); err != nil {
					fmt.Fprintln(os.Stderr, "tick:", err)
				}
			}
			if tick {
				fire()
				return nil
			}
			fmt.Println(i18n.Tf("iceflow daemon started (wakes at the next reminder, at most %s apart); press Ctrl-C to stop", interval))
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()
			for {
				fire()
				timer := time.NewTimer(a.nextWake(st, interval))
				select {
				case <-ctx.Done():
					timer.Stop()
					fmt.Println(i18n.T("iceflow daemon stopped"))
					return nil
				case <-timer.C:
				}
			}
		},
	}
	cmd.Flags().BoolVar(&tick, "tick", false, "run a single check and exit")
	cmd.Flags().DurationVar(&interval, "interval", time.Second, "maximum time to sleep between scheduler wake-ups")
	return cmd
}

// nextWake decides how long the resident scheduler sleeps. It returns the time
// until the earliest due reminder so reminders fire on time, capped at max so a
// reminder added while the daemon sleeps is still noticed promptly.
func (a *App) nextWake(st *store.Store, max time.Duration) time.Duration {
	const minWake = 50 * time.Millisecond
	next, ok, err := st.NextDueAt()
	if err != nil || !ok {
		return max
	}
	d := time.Until(next)
	if d < minWake {
		return minWake
	}
	if d > max {
		return max
	}
	return d
}

// tick fires every due reminder once and advances its schedule.
func (a *App) tick(st *store.Store) (int, error) {
	now := time.Now()
	due, err := st.DueReminders(now)
	if err != nil {
		return 0, err
	}
	notifier := notify.New(a.Cfg.Notify, a.Cfg.NotifySound)
	for _, r := range due {
		body := r.Notes
		if body == "" {
			body = i18n.T("iceflow reminder")
		}
		if err := notifier.Send(r.Title, body); err != nil {
			fmt.Fprintf(os.Stderr, "notify #%d: %v\n", r.ID, err)
		}
		fmt.Printf("[%s] 🔔 #%d %s\n", fmtLocal(now, "15:04:05"), r.ID, r.Title)
		if r.RRule != "" {
			if next := nextReminderTime(now, r.NextAt, r.RRule); !next.IsZero() {
				if err := st.UpdateReminderNext(r.ID, next); err != nil {
					return 0, err
				}
				continue
			}
		}
		if err := st.SetReminderEnabled(r.ID, false); err != nil {
			return 0, err
		}
	}
	return len(due), nil
}

func parseID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, errors.New(i18n.Tf("invalid id %q", s))
	}
	return id, nil
}

// nextReminderTime computes a reminder's next fire time. Remainders are stored
// in UTC, but recurrence must follow the local wall clock: "09:30" has to stay
// 09:30 locally across days, weeks and DST, and weekday/month boundaries are
// local, not UTC. Normalizing the anchor to the local zone first makes the
// recurrence math operate on the user's clock.
func nextReminderTime(now, nextAt time.Time, rrule string) time.Time {
	return recur.Next(now, nextAt.In(time.Local), rrule)
}

func joinArgs(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}
	return out
}
