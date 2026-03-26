package mc

type Options struct {
	Address     []string
	ServiceName string
}

type Option func(o *Options)

func WithAddress(address []string) Option {
	return func(o *Options) {
		o.Address = address
	}
}

func WithServiceName(serviceName string) Option {
	return func(o *Options) {
		o.ServiceName = serviceName
	}
}
