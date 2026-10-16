package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aice109/ice-flow-cli/internal/fuzzy"
	"github.com/aice109/ice-flow-cli/internal/i18n"
	"github.com/aice109/ice-flow-cli/internal/store"
)

type scoredSnippet struct {
	Snippet   store.Snippet
	Score     int
	CommandAt []int
}

func (a *App) findCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "find <query>",
		Short: "Fuzzy-search your command dictionary",
		Example: `  iceflow find doctor
  iceflow find reset
  iceflow find -n 5 list`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := a.openStore()
			if err != nil {
				return err
			}
			matches, err := a.search(st, strings.Join(args, " "))
			if err != nil {
				return err
			}
			if a.flagJSON {
				return printJSON(matchesJSON(matches))
			}
			if len(matches) == 0 {
				fmt.Println(dim(i18n.T("No matching commands.")))
				return nil
			}
			if limit > 0 && len(matches) > limit {
				matches = matches[:limit]
			}
			renderMatches(matches)
			return nil
		},
	}
	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "maximum results to show (0 = all)")
	return cmd
}

func (a *App) runCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "run <query>",
		Short: "Run the best matching command",
		Long:  i18n.T("Resolves the query against your command dictionary and executes the top match with `sh -c`."),
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := a.openStore()
			if err != nil {
				return err
			}
			matches, err := a.search(st, strings.Join(args, " "))
			if err != nil {
				return err
			}
			if len(matches) == 0 {
				return errors.New(i18n.Tf("no command matches %q", strings.Join(args, " ")))
			}
			top := matches[0].Snippet
			if !yes && !confirm(i18n.Tf("Run %q?", top.Command)) {
				return errors.New(i18n.T("aborted (use --yes to skip confirmation)"))
			}
			fmt.Fprintf(os.Stderr, "%s %s\n", dim("$"), top.Command)
			c := exec.Command("sh", "-c", top.Command)
			c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
			err = c.Run()
			_ = st.RecordSnippetUse(top.ID)
			if err != nil {
				return fmt.Errorf("%s: %w", i18n.T("command exited with error"), err)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation")
	return cmd
}

// search ranks snippets against the query using weighted fuzzy matching plus a
// popularity tiebreak.
func (a *App) search(st *store.Store, query string) ([]scoredSnippet, error) {
	all, err := st.ListSnippets()
	if err != nil {
		return nil, err
	}
	q := strings.TrimSpace(query)
	var out []scoredSnippet
	for _, sn := range all {
		res := scoredSnippet{Snippet: sn}
		if q == "" {
			res.Score = 1
			out = append(out, res)
			continue
		}
		matched := false
		best := 0
		if strings.EqualFold(sn.Command, q) {
			best = 100000
			matched = true
		}
		if m, ok := fuzzy.Match(q, sn.Command); ok {
			s := m.Score*3 + 60
			if s > best {
				best = s
				res.CommandAt = m.Positions
			}
			matched = true
		}
		if m, ok := fuzzy.Match(q, strings.Join(sn.Tags, " ")); ok {
			if s := m.Score*2 + 20; s > best {
				best = s
			}
			matched = true
		}
		if m, ok := fuzzy.Match(q, sn.Description); ok {
			if s := m.Score + 5; s > best {
				best = s
			}
			matched = true
		}
		if !matched {
			continue
		}
		res.Score = best + min(sn.UseCount, 20)*2
		out = append(out, res)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		if out[i].Snippet.UseCount != out[j].Snippet.UseCount {
			return out[i].Snippet.UseCount > out[j].Snippet.UseCount
		}
		return out[i].Snippet.ID < out[j].Snippet.ID
	})
	return out, nil
}

func renderMatches(matches []scoredSnippet) {
	hlPre, hlSuf := "", ""
	if colorEnabled {
		hlPre, hlSuf = "\x1b[1;33m", "\x1b[0m"
	}
	for _, m := range matches {
		sn := m.Snippet
		command := fuzzy.Highlight(sn.Command, m.CommandAt, hlPre, hlSuf)
		head := dim("#" + strconv.FormatInt(sn.ID, 10))
		if sn.UseCount > 0 {
			head += "  " + dim(fmt.Sprintf("%d×", sn.UseCount))
		}
		fmt.Println(head)
		fmt.Printf("    %s\n", bold(command))
		if sn.Description != "" {
			fmt.Printf("    %s\n", sn.Description)
		}
		if len(sn.Tags) > 0 {
			fmt.Printf("    %s\n", dim(i18n.T("tags: ")+strings.Join(sn.Tags, ", ")))
		}
		if sn.Example != "" {
			fmt.Printf("    %s %s\n", dim(i18n.T("example:")), sn.Example)
		}
		fmt.Println()
	}
}
