package asynq

import (
	"time"

	"github.com/realmicro/realmicro/broker"
)

const (
	// ClusterType means redis cluster.
	ClusterType = "cluster"
	// NodeType means redis node.
	NodeType = "node"

	DefaultDB = 0
)

type optionsKeyType struct{}
type publishKeyType struct{}
type subscribeKeyType struct{}

var (
	optionsKey   = optionsKeyType{}
	publishKey   = publishKeyType{}
	subscribeKey = subscribeKeyType{}
)

type brokerOptions struct {
	Type string
	Pass string
	DB   int
	tls  bool
}

// Cluster customizes the given Redis as a cluster
func Cluster() broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*brokerOptions)
		bo.Type = ClusterType
	}
}

func Pass(pass string) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*brokerOptions)
		bo.Pass = pass
	}
}

// DB is the redis db to use
func DB(db int) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*brokerOptions)
		bo.DB = db
	}
}

// Tls if open tls
func Tls(b bool) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*brokerOptions)
		bo.tls = b
	}
}

type publishOptions struct {
	Queue     string
	Opr       string
	ProcessIn time.Duration
	Retention time.Duration
}

func Queue(queue string) broker.PublishOption {
	return func(o *broker.PublishOptions) {
		po := o.Context.Value(publishKey).(*publishOptions)
		po.Queue = queue
	}
}

func Opr(opr string) broker.PublishOption {
	return func(o *broker.PublishOptions) {
		po := o.Context.Value(publishKey).(*publishOptions)
		po.Opr = opr
	}
}

func ProcessIn(t time.Duration) broker.PublishOption {
	return func(o *broker.PublishOptions) {
		po := o.Context.Value(publishKey).(*publishOptions)
		po.ProcessIn = t
	}
}

func Retention(t time.Duration) broker.PublishOption {
	return func(o *broker.PublishOptions) {
		po := o.Context.Value(publishKey).(*publishOptions)
		po.Retention = t
	}
}
