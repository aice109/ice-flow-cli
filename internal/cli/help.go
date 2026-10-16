package cli

import (
	"regexp"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/aice109/ice-flow-cli/internal/i18n"
)

// Command group IDs used to section the top-level help.
const (
	groupPlanning    = "planning"
	groupDictionary  = "dictionary"
	groupMaintenance = "maintenance"
	groupHelp        = "help"
)

func init() {
	// Expose the catalog to cobra templates so usage labels can be translated.
	cobra.AddTemplateFunc("t", i18n.T)
	cobra.AddTemplateFunc("tf", i18n.Tf)
}

// usageTemplate is cobra's default usage template with translatable labels.
const usageTemplate = `{{t "Usage:"}}{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

{{t "Aliases:"}}
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

{{t "Examples:"}}
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

{{t "Available Commands:"}}{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{$group.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

{{t "Additional Commands:"}}{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{t "Flags:"}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{t "Global Flags:"}}
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

{{t "Additional help topics:"}}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

{{tf "Use \"%s [command] --help\" for more information about a command." .CommandPath}}{{end}}
`

// addGroups sections the top-level command list in `iceflow --help`.
func addGroups(root *cobra.Command) {
	root.AddGroup(
		&cobra.Group{ID: groupPlanning, Title: i18n.T("Planning")},
		&cobra.Group{ID: groupDictionary, Title: i18n.T("Dictionary")},
		&cobra.Group{ID: groupMaintenance, Title: i18n.T("Maintenance")},
		&cobra.Group{ID: groupHelp, Title: i18n.T("Help")},
	)
	byName := map[string]string{
		"task":    groupPlanning,
		"remind":  groupPlanning,
		"cmd":     groupDictionary,
		"find":    groupDictionary,
		"run":     groupDictionary,
		"service": groupMaintenance,
		"update":  groupMaintenance,
		"config":  groupMaintenance,
		"init":    groupMaintenance,
		"backup":  groupMaintenance,
		"restore": groupMaintenance,
		"version": groupMaintenance,
	}
	for _, c := range root.Commands() {
		if g, ok := byName[c.Name()]; ok {
			c.GroupID = g
		}
	}
	root.SetHelpCommandGroupID(groupHelp)
	root.SetCompletionCommandGroupID(groupHelp)
}

// localizeTree translates command metadata (short descriptions and flag usage)
// in place. English strings are the catalog keys, so anything without a
// translation is left untouched.
//
// It also installs the -h/--help flag with a translated description, which
// stops cobra from adding its own English one.
func localizeTree(root *cobra.Command) {
	root.InitDefaultHelpCmd()
	root.InitDefaultCompletionCmd()

	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		c.Short = i18n.T(c.Short)
		c.Long = i18n.T(c.Long)

		translateFlag := func(f *pflag.Flag) {
			if f.Name == "help" {
				return
			}
			f.Usage = i18n.T(f.Usage)
		}
		c.Flags().VisitAll(translateFlag)
		c.PersistentFlags().VisitAll(translateFlag)
		c.InheritedFlags().VisitAll(translateFlag)

		if c.Flags().Lookup("help") == nil {
			c.Flags().BoolP("help", "h", false, i18n.T("show help for this command"))
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
}

var (
	reUnknownCommand    = regexp.MustCompile(`^unknown command "([^"]*)" for "([^"]*)"`)
	reUnknownShorthand  = regexp.MustCompile(`^unknown shorthand flag: (.*)$`)
	reUnknownFlag       = regexp.MustCompile(`^unknown flag: (.*)$`)
	reFlagNeedsArgument = regexp.MustCompile(`^flag needs an argument: (.*)$`)
	reAcceptsArgs       = regexp.MustCompile(`^accepts (\d+) arg\(s\), received (\d+)$`)
	reMinArgs           = regexp.MustCompile(`^requires at least (\d+) arg\(s\), only received (\d+)$`)
	reMaxArgs           = regexp.MustCompile(`^requires at most (\d+) arg\(s\), received (\d+)$`)
)

// localizeError translates the errors cobra generates for common situations
// (unknown command or flag, wrong argument count). Errors we create ourselves
// are already localized where they are raised.
func localizeError(err error) string {
	msg := err.Error()
	if i18n.Locale() == i18n.English {
		return msg
	}
	if msg == "not found" {
		return i18n.T("not found")
	}
	if m := reUnknownCommand.FindStringSubmatch(msg); m != nil {
		// Keep any trailing "Did you mean this?" suggestions.
		return i18n.Tf(`unknown command "%s" for "%s"`, m[1], m[2]) + msg[len(m[0]):]
	}
	for _, c := range []struct {
		re *regexp.Regexp
		en string
	}{
		{reUnknownShorthand, "unknown shorthand flag: %s"},
		{reUnknownFlag, "unknown flag: %s"},
		{reFlagNeedsArgument, "flag needs an argument: %s"},
	} {
		if m := c.re.FindStringSubmatch(msg); m != nil {
			return i18n.Tf(c.en, m[1])
		}
	}
	for _, c := range []struct {
		re *regexp.Regexp
		en string
	}{
		{reAcceptsArgs, "accepts %d arg(s), received %d"},
		{reMinArgs, "requires at least %d arg(s), only received %d"},
		{reMaxArgs, "requires at most %d arg(s), received %d"},
	} {
		if m := c.re.FindStringSubmatch(msg); m != nil {
			want, _ := strconv.Atoi(m[1])
			got, _ := strconv.Atoi(m[2])
			return i18n.Tf(c.en, want, got)
		}
	}
	return msg
}
