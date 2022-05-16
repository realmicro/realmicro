// Package loader manages loading from multiple sources
package loader

import (
	"context"

	"github.com/realmicro/realmicro/config/reader"
	"github.com/realmicro/realmicro/config/source"
)

// Loader manages loading sources
type Loader interface {
	// Close stop the loader
	Close() error
	// Load the sources
	Load(...source.Source) error
	// A Snapshot of loaded config
	Snapshot() (*Snapshot, error)
	// Sync force sync of sources
	Sync() error
	// Watch for changes
	Watch(...string) (Watcher, error)
	// String name of loader
	String() string
}

// Watcher lets you watch sources and returns a merged ChangeSet
type Watcher interface {
	// Next First call to next may return the current Snapshot
	// If you are watching a path then only the data from
	// that path is returned.
	Next() (*Snapshot, error)
	// Stop watching for changes
	Stop() error
}

// Snapshot is a merged ChangeSet
type Snapshot struct {
	// The merged ChangeSet
	ChangeSet *source.ChangeSet
	// Deterministic and comparable version of the snapshot
	Version string
}

type Options struct {
	Reader reader.Reader
	Source []source.Source

	// for alternative data
	Context context.Context

	WithWatcherDisabled bool
}

type Option func(o *Options)

// Copy snapshot
func Copy(s *Snapshot) *Snapshot {
	cs := *(s.ChangeSet)

	return &Snapshot{
		ChangeSet: &cs,
		Version:   s.Version,
	}
}
