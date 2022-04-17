package dc

import (
	"context"

	"github.com/realmicro/realmicro/client"
	"github.com/realmicro/realmicro/metadata"
	"github.com/realmicro/realmicro/registry"
	"github.com/realmicro/realmicro/selector"
)

type dcWrapper struct {
	client.Client
}

func (dc *dcWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	md, _ := metadata.FromContext(ctx)
	env, ok := md["Env"]
	if !ok || len(env) == 0 {
		env = "prod"
	}

	nOpts := append(opts, client.WithSelectOption(selector.WithFilter(func(services []*registry.Service) []*registry.Service {
		for _, service := range services {
			var nodes []*registry.Node
			for _, node := range service.Nodes {
				nodeEnv := node.Metadata["env"]
				if len(nodeEnv) == 0 {
					nodeEnv = "prod"
				}
				if env == nodeEnv {
					nodes = append(nodes, node)
				}
			}
			service.Nodes = nodes
		}
		return services
	})))
	return dc.Client.Call(ctx, req, rsp, nOpts...)
}

func NewDCWrapper(c client.Client) client.Client {
	return &dcWrapper{c}
}
