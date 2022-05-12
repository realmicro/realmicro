package etcd

import (
	"context"
	"fmt"
	"github.com/realmicro/realmicro/logger"
	"net"
	"time"

	"github.com/realmicro/realmicro/config/source"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/client/v3"
)

// Currently a single etcd reader
type etcd struct {
	prefix      string
	stripPrefix string
	ifCreate    bool
	opts        source.Options
	client      *clientv3.Client
	cerr        error
}

var (
	DefaultPrefix = "/real/config/"
)

func (c *etcd) Read() (*source.ChangeSet, error) {
	if c.cerr != nil {
		return nil, c.cerr
	}

	rsp, err := c.client.Get(context.Background(), c.prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	if rsp == nil || len(rsp.Kvs) == 0 {
		logger.Infof("rsp == nil || len(rsp.Kvs) == 0")
		if c.ifCreate {
			_, err = c.client.Put(context.Background(), c.prefix, "")
			if err != nil {
				return nil, err
			}
			rsp, err = c.client.Get(context.Background(), c.prefix, clientv3.WithPrefix())
			if err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("source not found: %s", c.prefix)
		}
	}

	kvs := make([]*mvccpb.KeyValue, 0, len(rsp.Kvs))
	for _, v := range rsp.Kvs {
		kvs = append(kvs, v)
	}

	data := makeMap(c.opts.Encoder, kvs, c.stripPrefix)
	logger.Infof("etcd data: %+v", data)

	b, err := c.opts.Encoder.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("error reading source: %v", err)
	}

	cs := &source.ChangeSet{
		Timestamp: time.Now(),
		Source:    c.String(),
		Data:      b,
		Format:    c.opts.Encoder.String(),
	}
	cs.Checksum = cs.Sum()

	return cs, nil
}

func (c *etcd) String() string {
	return "etcd"
}

func (c *etcd) Watch() (source.Watcher, error) {
	if c.cerr != nil {
		return nil, c.cerr
	}
	cs, err := c.Read()
	if err != nil {
		return nil, err
	}
	return newWatcher(c.prefix, c.stripPrefix, c.client.Watcher, cs, c.opts)
}

func (c *etcd) Write(cs *source.ChangeSet) error {
	return nil
}

func NewSource(opts ...source.Option) source.Source {
	options := source.NewOptions(opts...)

	var endpoints []string

	// check if there are any addrs
	addrs, ok := options.Context.Value(addressKey{}).([]string)
	if ok {
		for _, a := range addrs {
			addr, port, err := net.SplitHostPort(a)
			if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
				port = "2379"
				addr = a
				endpoints = append(endpoints, fmt.Sprintf("%s:%s", addr, port))
			} else if err == nil {
				endpoints = append(endpoints, fmt.Sprintf("%s:%s", addr, port))
			}
		}
	}

	if len(endpoints) == 0 {
		endpoints = []string{"localhost:2379"}
	}

	// check dial timeout option
	dialTimeout, ok := options.Context.Value(dialTimeoutKey{}).(time.Duration)
	if !ok {
		dialTimeout = 3 * time.Second // default dial timeout
	}

	config := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: dialTimeout,
	}

	u, ok := options.Context.Value(authKey{}).(*authCredit)
	if ok {
		config.Username = u.Username
		config.Password = u.Password
	}

	// use default config
	client, err := clientv3.New(config)

	prefix := DefaultPrefix
	sp := ""
	f, ok := options.Context.Value(prefixKey{}).(string)
	if ok {
		prefix = DefaultPrefix + f
	}

	if b, ok := options.Context.Value(stripPrefixKey{}).(bool); ok && b {
		sp = prefix
	}

	var ifCreate bool
	ifCreate, _ = options.Context.Value(prefixCreateKey{}).(bool)

	return &etcd{
		prefix:      prefix,
		stripPrefix: sp,
		ifCreate:    ifCreate,
		opts:        options,
		client:      client,
		cerr:        err,
	}
}
