// Package handler provides http handlers
package handler

import (
	"net/http"
)

// Handler represents a HTTP handler that manages a request.
type Handler interface {
	// Handler standard http handler
	http.Handler
	// String name of handler
	String() string
}
