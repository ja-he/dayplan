package cli

type DayplanIntelServerCommandLineOpts struct {
	RunCommand     DayplanIntelServerRunCommand `command:"run" subcommands-optional:"true"`
	VersionCommand VersionCommand               `command:"version" subcommands-optional:"true"`
}
