package cli

type CommandLineOpts struct {
	Day     string `short:"d" long:"day" description:"Specify the day to plan" value-name:"<file>"`
	Version bool   `short:"v" long:"version" description:"Show the program version"`

	SummarizeCommand SummarizeCommand `command:"summarize" subcommands-optional:"true"`
	VersionCommand   VersionCommand   `command:"version" subcommands-optional:"true"`
}

var Opts CommandLineOpts
