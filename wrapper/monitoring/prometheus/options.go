package prometheus

type Options struct {
	Name    string
	Version string
	ID      string
}

type Option func(*Options)

func ServiceName(name string) Option {
	return func(opts *Options) {
		opts.Name = name
	}
}

func ServiceVersion(version string) Option {
	return func(opts *Options) {
		opts.Version = version
	}
}

func ServiceID(id string) Option {
	return func(opts *Options) {
		opts.ID = id
	}
}
