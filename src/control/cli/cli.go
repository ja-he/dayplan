package cli

type CommandLineOpts struct {
	Day     string `short:"d" long:"day" description:"Specify the day to plan" value-name:"<file>"`
	Version bool   `short:"v" long:"version" description:"Show the program version"`
	Theme   string `short:"t" long:"theme" choice:"light" choice:"dark" description:"Select a 'dark' or a 'light' default theme (note: only sets defaults, which are individually overridden by settings in config.yaml"`

	SummarizeCommand SummarizeCommand `command:"summarize" subcommands-optional:"true"`
	VersionCommand   VersionCommand   `command:"version" subcommands-optional:"true"`
}

var Opts CommandLineOpts
