package command

import (
	"fmt"

	"github.com/realmicro/realmicro/registry"
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

			for _, svc := range services {
				fmt.Printf("service: %s\n", svc.Name)
				if svc.Version != "" {
					fmt.Printf("version: %s\n", svc.Version)
				}

				fmt.Println()

				// Print nodes
				if len(svc.Nodes) > 0 {
					fmt.Println("nodes:")
					for _, node := range svc.Nodes {
						fmt.Printf("  - %s %s\n", node.Id, node.Address)
					}
					fmt.Println()
				}

				// Print endpoints
				if len(svc.Endpoints) > 0 {
					fmt.Println("endpoints:")
					for _, ep := range svc.Endpoints {
						fmt.Printf("  %s\n", ep.Name)
						if ep.Request != nil {
							fmt.Printf("    Request: %s\n", ep.Request.Name)
							printValues(ep.Request.Values, 6)
						}
						if ep.Response != nil {
							fmt.Printf("    Response: %s\n", ep.Response.Name)
							printValues(ep.Response.Values, 6)
						}
						fmt.Println()
					}
				}
			}

			return nil
		},
	}
}

func printValues(values []*registry.Value, indent int) {
	for _, v := range values {
		fmt.Printf("%*s- %s %s\n", indent, "", v.Name, v.Type)
		if len(v.Values) > 0 {
			printValues(v.Values, indent+2)
		}
	}
}
