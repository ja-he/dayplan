package cli

type DayplanCommandLineOpts struct {
	Version bool `short:"v" long:"version" description:"Show the program version"`

	TuiCommand       DayplanTUICommand       `command:"tui" subcommands-optional:"true"`
	SummarizeCommand DayplanSummarizeCommand `command:"summarize" subcommands-optional:"true"`
	TimesheetCommand DayplanTimesheetCommand `command:"timesheet" subcommands-optional:"true"`
	AddCommand       DayplanAddCommand       `command:"add" subcommands-optional:"true"`

	VersionCommand VersionCommand `command:"version" subcommands-optional:"true"`
}

var DayplanOpts DayplanCommandLineOpts
