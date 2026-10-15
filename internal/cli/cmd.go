package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/aice109/ice-flow-cli/internal/i18n"
	"github.com/aice109/ice-flow-cli/internal/store"
)

func (a *App) snippetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "cmd",
		Aliases: []string{"cmds", "snippet"},
		Short:   "Manage your command dictionary",
		Long: i18n.T("Store commands together with an explanation so you can recall them later.") + "\n\n" +
			i18n.T("Search them with `iflow find <query>` and run one with `iflow run <query>`."),
	}
	cmd.AddCommand(
		a.snippetAddCmd(),
		a.snippetListCmd(),
		a.snippetEditCmd(),
		a.snippetRmCmd(),
		a.snippetImportCmd(),
		a.snippetExportCmd(),
	)
	return cmd
}

func (a *App) snippetAddCmd() *cobra.Command {
	var desc, example, shell string
	var tags []string
	cmd := &cobra.Command{
		Use:   "add <command>",
		Short: "Add a command with an explanation",
		Example: `  iflow cmd add "flutter doctor" -d "Check the Flutter environment" -t flutter,env
  iflow cmd add "git reset --hard HEAD~1" -d "Drop the last commit (dangerous)"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := a.openStore()
			if err != nil {
				return err
			}
			sn := &store.Snippet{
				Command:     joinArgs(args),
				Description: desc,
				Tags:        tags,
				Example:     example,
				Shell:       shell,
			}
			id, err := st.CreateSnippet(sn)
			if err != nil {
				return err
			}
			fmt.Println(i18n.Tf("added command %s", bold("#"+strconv.FormatInt(id, 10))))
			return nil
		},
	}
	cmd.Flags().StringVarP(&desc, "desc", "d", "", "what the command does")
	cmd.Flags().StringSliceVarP(&tags, "tag", "t", nil, "tags (repeatable or comma-separated)")
	cmd.Flags().StringVar(&example, "example", "", "an example invocation")
	cmd.Flags().StringVar(&shell, "shell", "", "target shell (bash, zsh, ...)")
	return cmd
}

func (a *App) snippetListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List stored commands",
		Example: `  iflow cmd list
  iflow cmd list --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := a.openStore()
			if err != nil {
				return err
			}
			snippets, err := st.ListSnippets()
			if err != nil {
				return err
			}
			if a.flagJSON {
				return printJSON(snippetsJSON(snippets))
			}
			if len(snippets) == 0 {
				fmt.Println(dim(i18n.T("No commands yet. Add one with `iflow cmd add`.")))
				return nil
			}
			rows := make([][]string, 0, len(snippets))
			for _, sn := range snippets {
				rows = append(rows, []string{
					strconv.FormatInt(sn.ID, 10),
					fmt.Sprintf("%d×", sn.UseCount),
					bold(sn.Command),
					sn.Description,
					tagsLabel(sn.Tags),
				})
			}
			printTable([]string{dim(i18n.T("ID")), dim(i18n.T("Uses")), dim(i18n.T("Command")), dim(i18n.T("Description")), dim(i18n.T("Tags"))}, rows)
			return nil
		},
	}
}

func (a *App) snippetEditCmd() *cobra.Command {
	var command, desc, example, shell string
	var tags []string
	cmd := &cobra.Command{
		Use:     "edit <id>",
		Short:   "Edit a stored command",
		Example: `  iflow cmd edit 4 -d "updated description" -t git`,
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
			sn, err := st.GetSnippet(id)
			if err != nil {
				return err
			}
			if command != "" {
				sn.Command = command
			}
			if desc != "" {
				sn.Description = desc
			}
			if example != "" {
				sn.Example = example
			}
			if shell != "" {
				sn.Shell = shell
			}
			if len(tags) > 0 {
				sn.Tags = tags
			}
			if err := st.UpdateSnippet(sn); err != nil {
				return err
			}
			fmt.Println(i18n.Tf("updated command #%d", id))
			return nil
		},
	}
	cmd.Flags().StringVar(&command, "command", "", "replace the command text")
	cmd.Flags().StringVarP(&desc, "desc", "d", "", "replace the description")
	cmd.Flags().StringSliceVarP(&tags, "tag", "t", nil, "replace tags")
	cmd.Flags().StringVar(&example, "example", "", "replace the example")
	cmd.Flags().StringVar(&shell, "shell", "", "replace the shell")
	return cmd
}

func (a *App) snippetRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "rm <id>",
		Aliases: []string{"delete", "remove"},
		Short:   "Delete a stored command",
		Example: "  iflow cmd rm 4",
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
			if err := st.DeleteSnippet(id); err != nil {
				return err
			}
			fmt.Println(i18n.Tf("deleted command #%d", id))
			return nil
		},
	}
}

func (a *App) snippetImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <file>",
		Short: "Import commands from a JSON or Markdown file",
		Long: i18n.T("Import commands from a file. Two formats are accepted:") + "\n\n" +
			"JSON:  [{\"command\":\"flutter doctor\",\"description\":\"...\",\"tags\":[\"flutter\"],\"example\":\"...\"}]\n" +
			i18n.T("Text:  one command per line, \"command :: description :: tag1,tag2\"") + "\n" +
			i18n.T("       Markdown list items like \"- `command` - description\" also work."),
		Example: `  iflow cmd import ~/notes.md
  iflow cmd import commands.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			snippets, err := parseSnippets(data)
			if err != nil {
				return err
			}
			st, err := a.openStore()
			if err != nil {
				return err
			}
			added := 0
			for _, sn := range snippets {
				if _, err := st.CreateSnippet(&sn); err != nil {
					return err
				}
				added++
			}
			fmt.Println(i18n.Tf("imported %d commands", added))
			return nil
		},
	}
}

func (a *App) snippetExportCmd() *cobra.Command {
	var format string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export commands as JSON or Markdown",
		Example: `  iflow cmd export --format md > commands.md
  iflow cmd export -f json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := a.openStore()
			if err != nil {
				return err
			}
			snippets, err := st.ListSnippets()
			if err != nil {
				return err
			}
			switch strings.ToLower(format) {
			case "json":
				return printJSON(snippetsJSON(snippets))
			case "md", "markdown":
				fmt.Println(i18n.T("# iflow command dictionary"))
				fmt.Println()
				for _, sn := range snippets {
					fmt.Printf("- `%s`", sn.Command)
					if sn.Description != "" {
						fmt.Printf(" — %s", sn.Description)
					}
					fmt.Println()
					if len(sn.Tags) > 0 {
						fmt.Println(i18n.Tf("  - tags: %s", strings.Join(sn.Tags, ", ")))
					}
					if sn.Example != "" {
						fmt.Println(i18n.Tf("  - example: `%s`", sn.Example))
					}
				}
				return nil
			default:
				return errors.New(i18n.Tf("unknown format %q (use json or md)", format))
			}
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", "md", "output format: json|md")
	return cmd
}

var mdListItemRe = regexp.MustCompile("^\\s*[-*]\\s+`([^`]+)`\\s*(?:[-—:]\\s*)?(.*)$")

// parseSnippets decodes JSON, "command :: description :: tags" lines, or
// Markdown list items.
func parseSnippets(data []byte) ([]store.Snippet, error) {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
		if strings.HasPrefix(trimmed, "{") {
			var one store.Snippet
			if err := json.Unmarshal([]byte(trimmed), &one); err != nil {
				return nil, err
			}
			return []store.Snippet{one}, nil
		}
		var many []store.Snippet
		if err := json.Unmarshal([]byte(trimmed), &many); err != nil {
			return nil, err
		}
		return many, nil
	}

	var out []store.Snippet
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if m := mdListItemRe.FindStringSubmatch(line); m != nil {
			out = append(out, store.Snippet{Command: strings.TrimSpace(m[1]), Description: strings.TrimSpace(m[2])})
			continue
		}
		parts := strings.Split(line, "::")
		sn := store.Snippet{Command: strings.TrimSpace(parts[0])}
		if len(parts) > 1 {
			sn.Description = strings.TrimSpace(parts[1])
		}
		if len(parts) > 2 {
			sn.Tags = store.SplitTags(parts[2])
		}
		if sn.Command != "" {
			out = append(out, sn)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Command < out[j].Command })
	return out, nil
}
