package cli

type CommandLineOpts struct {
	Day string `short:"d" long:"day" description:"Specify the day to plan" value-name:"<file>"`

	SummarizeCommand SummarizeCommand `command:"summarize" subcommands-optional:"true"`
}

var Opts CommandLineOpts
