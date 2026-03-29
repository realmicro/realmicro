package command

import (
	"github.com/realmicro/realmicro/registry"
	"github.com/urfave/cli/v2"
)

// NewRegistry is set by main to create a registry from CLI flags.
var NewRegistry func(c *cli.Context) registry.Registry

func getRegistry(c *cli.Context) registry.Registry {
	if NewRegistry != nil {
		return NewRegistry(c)
	}
	return registry.NewRegistry()
}
