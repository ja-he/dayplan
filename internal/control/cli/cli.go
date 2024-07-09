// Package cli provides the command-line interface for dayplan.
package cli

type CommandLineOpts struct {
	Version bool `short:"v" long:"version" description:"Show the program version"`

	TuiCommand       TUICommand       `command:"tui" subcommands-optional:"true"`
	SummarizeCommand SummarizeCommand `command:"summarize" subcommands-optional:"true"`
	TimesheetCommand TimesheetCommand `command:"timesheet" subcommands-optional:"true"`
	AddCommand       AddCommand       `command:"add" subcommands-optional:"true"`
	VersionCommand   VersionCommand   `command:"version" subcommands-optional:"true"`
}

var Opts CommandLineOpts
