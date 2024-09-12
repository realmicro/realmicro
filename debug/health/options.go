package health

var (
	DefaultAddress = ":6161"
)

// Option used by the health
type Option func(*Options)

// Options are logger options
type Options struct {
	Address string
}

// Address of the health
func Address(a string) Option {
	return func(o *Options) {
		o.Address = a
	}
}
