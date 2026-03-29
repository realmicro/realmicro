package command

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func Describe() *cli.Command {
	return &cli.Command{
		Name:      "describe",
		Usage:     "Describe a service and its endpoints",
		ArgsUsage: "<service>",
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return fmt.Errorf("service name required")
			}

			name := c.Args().First()
			reg := getRegistry(c)

			services, err := reg.GetService(name)
			if err != nil {
				return fmt.Errorf("failed to get service %s: %w", name, err)
			}

			if len(services) == 0 {
				return fmt.Errorf("service %s not found", name)
			}

			if len(services) == 1 {
				return printJSON(services[0])
			}
			return printJSON(services)
		},
	}
}
