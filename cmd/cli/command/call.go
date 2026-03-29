package command

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/realmicro/realmicro/client"
	"github.com/realmicro/realmicro/selector"
	"github.com/urfave/cli/v2"
)

func Call() *cli.Command {
	return &cli.Command{
		Name:      "call",
		Usage:     "Call a service endpoint",
		ArgsUsage: "<service> <endpoint> [json]",
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				return fmt.Errorf("usage: realmicro call <service> <endpoint> [json]")
			}

			service := c.Args().Get(0)
			endpoint := c.Args().Get(1)

			var body json.RawMessage
			if c.NArg() > 2 {
				body = json.RawMessage(c.Args().Get(2))
				if !json.Valid(body) {
					return fmt.Errorf("invalid JSON: %s", c.Args().Get(2))
				}
			} else {
				body = json.RawMessage("{}")
			}

			reg := getRegistry(c)
			sel := selector.NewSelector(selector.Registry(reg))

			cl := client.NewClient(
				client.Registry(reg),
				client.Selector(sel),
			)

			req := cl.NewRequest(service, endpoint, &body, client.WithContentType("application/json"))

			var rsp json.RawMessage
			if err := cl.Call(context.Background(), req, &rsp); err != nil {
				return fmt.Errorf("call error: %w", err)
			}

			var out interface{}
			if err := json.Unmarshal(rsp, &out); err != nil {
				fmt.Println(string(rsp))
				return nil
			}

			return printJSON(out)
		},
	}
}
