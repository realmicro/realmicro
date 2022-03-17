package redis

type Options struct {
	Type  string
	Addrs []string
	Pass  string
	DB    int
	tls   bool
}

type Option func(*Options)

// Cluster customizes the given Redis as a cluster
func Cluster() Option {
	return func(o *Options) {
		o.Type = ClusterType
	}
}

// Addrs the redis address to use
func Addrs(addrs []string) Option {
	return func(o *Options) {
		o.Addrs = addrs
	}
}

// Pass is the redis password to use
func Pass(pass string) Option {
	return func(o *Options) {
		o.Pass = pass
	}
}

// DB is the redis db to use
func DB(db int) Option {
	return func(o *Options) {
		o.DB = db
	}
}

// Tls if open tls
func Tls(b bool) Option {
	return func(o *Options) {
		o.tls = b
	}
}
