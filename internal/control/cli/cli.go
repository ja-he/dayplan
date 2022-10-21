package cli

type CommandLineOpts struct {
	Version bool `short:"v" long:"version" description:"Show the program version"`

	TuiCommand       TuiCommand       `command:"tui" subcommands-optional:"true"`
	SummarizeCommand SummarizeCommand `command:"summarize" subcommands-optional:"true"`
	AddCommand       AddCommand       `command:"add" subcommands-optional:"true"`
	VersionCommand   VersionCommand   `command:"version" subcommands-optional:"true"`
}

var Opts CommandLineOpts
