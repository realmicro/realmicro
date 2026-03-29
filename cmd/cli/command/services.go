package command

import (
	"fmt"
	"sort"

	"github.com/urfave/cli/v2"
)

func Services() *cli.Command {
	return &cli.Command{
		Name:  "services",
		Usage: "List all registered services",
		Action: func(c *cli.Context) error {
			reg := getRegistry(c)

			services, err := reg.ListServices()
			if err != nil {
				return fmt.Errorf("failed to list services: %w", err)
			}

			names := make([]string, 0, len(services))
			for _, s := range services {
				names = append(names, s.Name)
			}
			sort.Strings(names)

			return printJSON(names)
		},
	}
}
