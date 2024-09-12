package health

type Health interface {
	// Start the health
	Start() error
	// Stop the health
	Stop() error
	// String name of the health
	String() string
}

var (
	DefaultHealth Health = new(noop)
)

type noop struct{}

func (p *noop) Start() error {
	return nil
}

func (p *noop) Stop() error {
	return nil
}

func (p *noop) String() string {
	return "noop"
}
