package transport

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/realmicro/realmicro/logger"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type httpTransportListener struct {
	ht       *httpTransport
	listener net.Listener
}

func (h *httpTransportListener) Addr() string {
	return h.listener.Addr().String()
}

func (h *httpTransportListener) Close() error {
	return h.listener.Close()
}

func (h *httpTransportListener) Accept(fn func(Socket)) error {
	// create handler mux
	mux := http.NewServeMux()

	// register our transport handler
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var buf *bufio.ReadWriter
		var con net.Conn

		// read a regular request
		if r.ProtoMajor == 1 {
			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			r.Body = ioutil.NopCloser(bytes.NewReader(b))

			// Hijack to conn
			// We also don't close the connection here, as it will be closed by
			// the httpTransportSocket
			hj, ok := w.(http.Hijacker)
			if !ok {
				// we're screwed
				http.Error(w, "cannot serve conn", http.StatusInternalServerError)
				return
			}

			conn, bufrw, err := hj.Hijack()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer func() {
				if err := conn.Close(); err != nil {
					logger.Logf(logger.ErrorLevel, "Failed to close TCP connection: %v", err)
				}
			}()

			buf = bufrw
			con = conn
		}

		// buffered reader
		bufr := bufio.NewReader(r.Body)

		// save the request
		ch := make(chan *http.Request, 1)
		ch <- r

		// create new transport socket
		sock := &httpTransportSocket{
			ht:     h.ht,
			w:      w,
			r:      r,
			rw:     buf,
			buf:    bufr,
			ch:     ch,
			conn:   con,
			local:  h.Addr(),
			remote: r.RemoteAddr,
			closed: make(chan bool),
		}

		// execute the socket
		fn(sock)
	})

	// get optional handlers
	if h.ht.opts.Context != nil {
		handlers, ok := h.ht.opts.Context.Value("http_handlers").(map[string]http.Handler)
		if ok {
			for pattern, handler := range handlers {
				mux.Handle(pattern, handler)
			}
		}
	}

	// default http2 server
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: time.Second * 5,
	}

	// insecure connection use h2c
	if !(h.ht.opts.Secure || h.ht.opts.TLSConfig != nil) {
		srv.Handler = h2c.NewHandler(mux, &http2.Server{})
	}

	return srv.Serve(h.listener)
}
