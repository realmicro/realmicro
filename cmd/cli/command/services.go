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

			sort.Slice(services, func(i, j int) bool {
				return services[i].Name < services[j].Name
			})

			for _, s := range services {
				fmt.Println(s.Name)
			}

			return nil
		},
	}
}
