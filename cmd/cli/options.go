package cli

import (
	"context"

	"github.com/realmicro/realmicro/auth"
	"github.com/realmicro/realmicro/debug/profile"
	"github.com/realmicro/realmicro/registry"
	"github.com/realmicro/realmicro/selector"
)

type Option func(o *Options)

type Options struct {

	// Other options for implementations of the interface
	// can be stored in a context
	Context      context.Context
	Auth         *auth.Auth
	Selector     *selector.Selector
	DebugProfile *profile.Profile

	Registry *registry.Registry
}
