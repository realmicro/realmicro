package wrapper

import (
	"context"
	"strings"
	"time"

	"github.com/realmicro/realmicro/client"
	"github.com/realmicro/realmicro/errors"
	"github.com/realmicro/realmicro/logger"
	"github.com/realmicro/realmicro/metadata"
	"github.com/realmicro/realmicro/server"
	"github.com/sirupsen/logrus"
)

type logWrapper struct {
	client.Client
}

func (c *logWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	md, _ := metadata.FromContext(ctx)
	traceId := md["X-Trace-Id"]
	startTime := time.Now()
	var err error
	defer func() {
		logger.Fields(logrus.Fields{
			"TraceId":  traceId,
			"Service":  req.Service(),
			"Method":   req.Endpoint(),
			"Request":  req.Body(),
			"Code":     errors.Code(err),
			"Duration": time.Since(startTime) / time.Millisecond,
		}).Log(logger.InfoLevel, "ClientCall")
	}()
	err = c.Client.Call(ctx, req, rsp, opts...)
	return err
}

// LogCall is a call log wrapper
func LogCall(c client.Client) client.Client {
	return &logWrapper{c}
}

// LogHandler wraps a server handler to log
func LogHandler() server.HandlerWrapper {
	// return a handler wrapper
	return func(h server.HandlerFunc) server.HandlerFunc {
		// return a function that returns a function
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			md, _ := metadata.FromContext(ctx)
			traceId := md["X-Trace-Id"]
			realIp := md["X-Real-Ip"]
			platform := strings.ToLower(md["X-Platform"])
			version := md["X-Version"]
			project := md["X-Annie"]
			startTime := time.Now()
			var err error
			defer func() {
				logger.Fields(logrus.Fields{
					"TraceId":  traceId,
					"RemoteIp": realIp,
					"Platform": platform,
					"Project":  project,
					"Version":  version,
					"Service":  req.Service(),
					"Method":   req.Endpoint(),
					"Request":  req.Body(),
					"Code":     errors.Code(err),
					"Duration": time.Since(startTime) / time.Millisecond,
				}).Log(logger.InfoLevel, "ServiceHandler")
			}()
			err = h(ctx, req, rsp)
			return err
		}
	}
}
