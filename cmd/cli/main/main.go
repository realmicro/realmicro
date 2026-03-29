package main

import (
	"fmt"
	"os"

	"github.com/realmicro/realmicro/cmd/cli/command"
	"github.com/realmicro/realmicro/registry"
	"github.com/realmicro/realmicro/registry/etcd"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "realmicro",
		Usage: "A command line interface for RealMicro",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "reg",
				Usage: "Registry type (etcd, mdns)",
				Value: "etcd",
			},
			&cli.StringFlag{
				Name:  "reghost",
				Usage: "Registry address",
			},
		},
		Commands: []*cli.Command{
			command.Services(),
			command.Describe(),
			command.Call(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// NewRegistry creates a registry based on CLI flags.
// Exported so command package can use it via app metadata.
func init() {
	command.NewRegistry = func(c *cli.Context) registry.Registry {
		regType := c.String("reg")
		regHost := c.String("reghost")

		switch regType {
		case "etcd":
			var opts []registry.Option
			if regHost != "" {
				opts = append(opts, registry.Addrs(regHost))
			}
			return etcd.NewRegistry(opts...)
		default:
			var opts []registry.Option
			if regHost != "" {
				opts = append(opts, registry.Addrs(regHost))
			}
			return registry.NewRegistry(opts...)
		}
	}
}
