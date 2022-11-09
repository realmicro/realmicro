package wrapper

import (
	"context"

	"github.com/realmicro/realmicro/client"
	"github.com/realmicro/realmicro/logger"
	"github.com/realmicro/realmicro/metadata"
	"github.com/realmicro/realmicro/registry"
	"github.com/realmicro/realmicro/selector"
	"github.com/sirupsen/logrus"
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

	nOpts := append(opts, client.WithSelectOption(selector.WithFilter(func(old []*registry.Service) []*registry.Service {
		var services []*registry.Service

		for _, service := range old {
			serv := new(registry.Service)
			var nodes []*registry.Node
			for _, node := range service.Nodes {
				if node.Metadata == nil {
					continue
				}
				nodeEnv := node.Metadata["env"]
				if len(nodeEnv) == 0 {
					nodeEnv = "prod"
				}
				if env == nodeEnv {
					nodes = append(nodes, node)
				}
			}
			if len(nodes) == 0 {
				ss, err := registry.GetService(service.Name)
				if err != nil {
					logger.Fields(logrus.Fields{
						"Wrapper":       "DcFilter",
						"GetServiceErr": err,
					}).Log(logger.ErrorLevel, "ServiceNodeNoneAvailable")
				} else {
					for _, s := range ss {
						logger.Fields(logrus.Fields{
							"Wrapper":        "DcFilter",
							"ServiceName":    s.Name,
							"ServiceVersion": s.Version,
							"ServiceNodes":   s.Nodes,
						}).Log(logger.ErrorLevel, "ServiceNodeNoneAvailable")
					}
				}
				logger.Fields(logrus.Fields{
					"Wrapper":      "DcFilter",
					"Env":          env,
					"Metadata":     md,
					"ServiceNodes": service.Nodes,
				}).Log(logger.ErrorLevel, "ServiceNodeNoneAvailable")
			}
			// copy
			*serv = *service
			serv.Nodes = nodes
			services = append(services, serv)
		}
		return services
	})))
	return dc.Client.Call(ctx, req, rsp, nOpts...)
}

func NewDCWrapper(c client.Client) client.Client {
	return &dcWrapper{c}
}
