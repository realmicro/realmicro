package asynq

import "github.com/realmicro/realmicro/broker"

const (
	// ClusterType means redis cluster.
	ClusterType = "cluster"
	// NodeType means redis node.
	NodeType = "node"

	DefaultDB = 0
)

type optionsKeyType struct{}

var (
	optionsKey = optionsKeyType{}
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
