package realbreaker

import (
	"context"
	"sync"

	"github.com/realmicro/realmicro/client"
	"github.com/realmicro/realmicro/common/util/breaker"
	"github.com/realmicro/realmicro/errors"
)

type BreakerMethod int

const (
	BreakService BreakerMethod = iota
	BreakServiceEndpoint
)

type clientWrapper struct {
	bm BreakerMethod
	bs map[string]breaker.Breaker
	mu sync.Mutex
	client.Client
}

func (c *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	var svc string

	switch c.bm {
	case BreakService:
		svc = req.Service()
	case BreakServiceEndpoint:
		svc = req.Service() + "." + req.Endpoint()
	}

	c.mu.Lock()
	brk, ok := c.bs[svc]
	if !ok {
		brk = breaker.New(breaker.WithName(svc))
		c.bs[svc] = brk
	}
	c.mu.Unlock()

	promise, err := brk.Allow()
	if err != nil {
		return errors.New(req.Service(), err.Error(), 502)
	}

	if err = c.Client.Call(ctx, req, rsp, opts...); err == nil {
		promise.Accept()
		return nil
	}

	merr := errors.Parse(err.Error())
	switch {
	case merr.Code == 0:
		merr.Code = 503
	case len(merr.Id) == 0:
		merr.Id = req.Service()
	}

	if merr.Code >= 500 {
		promise.Reject(merr.Error())
	} else {
		promise.Accept()
	}

	return merr
}

// NewClientWrapper returns a client Wrapper.
func NewClientWrapper() client.Wrapper {
	return func(c client.Client) client.Client {
		w := &clientWrapper{}
		w.bm = BreakService
		w.bs = make(map[string]breaker.Breaker)
		w.Client = c
		return w
	}
}
