package cli

import "github.com/urfave/cli/v2"

type Cmd interface {
	// App The cli app within this cmd
	App() *cli.App
	// Init Adds options, parses flags and initialize exits on error
	Init(opts ...Option) error
	// Options set within this command
	Options() Options
}
