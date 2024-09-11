package transport

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/realmicro/realmicro/codec"
)

var (
	DefaultBufSizeH2 = 4 * 1024 * 1024
)

type Options struct {
	// Codec is the codec interface to use where headers are not supported
	// by the transport and the entire payload must be encoded
	Codec codec.Marshaler
	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
	// TLSConfig to secure the connection. The assumption is that this
	// is mTLS keypair
	TLSConfig *tls.Config
	// Addrs is the list of intermediary addresses to connect to
	Addrs []string
	// Timeout sets the timeout for Send/Recv
	Timeout time.Duration
	// BuffSizeH2 is the HTTP2 buffer size
	BuffSizeH2 int
	// Secure tells the transport to secure the connection.
	// In the case TLSConfig is not specified the best effort self-signed
	// certs should be used
	Secure bool
}

type DialOptions struct {
	// TODO: add tls options when dialling
	// Currently set in global options

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
	// Timeout for dialing
	Timeout time.Duration
	// Tells the transport this is a streaming connection with
	// multiple calls to send/recv and that send may not even be called
	Stream bool
	// ConnClose sets the Connection header to close
	ConnClose bool
	// InsecureSkipVerify skip TLS verification.
	InsecureSkipVerify bool
}

type ListenOptions struct {
	// TODO: add tls options when listening
	// Currently set in global options

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

// Addrs to use for transport
func Addrs(addrs ...string) Option {
	return func(o *Options) {
		o.Addrs = addrs
	}
}

// Codec sets the codec used for encoding where the transport
// does not support message headers
func Codec(c codec.Marshaler) Option {
	return func(o *Options) {
		o.Codec = c
	}
}

// Timeout sets the timeout for Send/Recv execution
func Timeout(t time.Duration) Option {
	return func(o *Options) {
		o.Timeout = t
	}
}

// Secure Use secure communication. If TLSConfig is not specified we
// use InsecureSkipVerify and generate a self-signed cert
func Secure(b bool) Option {
	return func(o *Options) {
		o.Secure = b
	}
}

// TLSConfig to be used for the transport.
func TLSConfig(t *tls.Config) Option {
	return func(o *Options) {
		o.TLSConfig = t
	}
}

// WithStream Indicates whether this is a streaming connection
func WithStream() DialOption {
	return func(o *DialOptions) {
		o.Stream = true
	}
}

// WithTimeout Timeout used when dialling the remote side
func WithTimeout(d time.Duration) DialOption {
	return func(o *DialOptions) {
		o.Timeout = d
	}
}

// WithConnClose sets the Connection header to close.
func WithConnClose() DialOption {
	return func(o *DialOptions) {
		o.ConnClose = true
	}
}

func WithInsecureSkipVerify(b bool) DialOption {
	return func(o *DialOptions) {
		o.InsecureSkipVerify = b
	}
}

// BuffSizeH2 sets the HTTP2 buffer size.
// Default is 4 * 1024 * 1024.
func BuffSizeH2(size int) Option {
	return func(o *Options) {
		o.BuffSizeH2 = size
	}
}

// NetListener InsecureSkipVerify sets the TLS options to skip verification.
// Set net.Listener for httpTransport.
func NetListener(customListener net.Listener) ListenOption {
	return func(o *ListenOptions) {
		if customListener == nil {
			return
		}

		if o.Context == nil {
			o.Context = context.TODO()
		}

		o.Context = context.WithValue(o.Context, netListener{}, customListener)
	}
}
