package transport

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type httpTransportSocket struct {
	w http.ResponseWriter

	// the hijacked when using http 1
	conn net.Conn
	ht   *httpTransport
	r    *http.Request
	rw   *bufio.ReadWriter

	// for the first request
	ch chan *http.Request

	// h2 things
	buf *bufio.Reader
	// indicate if socket is closed
	closed chan bool

	// local/remote ip
	local  string
	remote string

	mtx sync.RWMutex
}

func (h *httpTransportSocket) Local() string {
	return h.local
}

func (h *httpTransportSocket) Remote() string {
	return h.remote
}

func (h *httpTransportSocket) Recv(m *Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}

	if m.Header == nil {
		m.Header = make(map[string]string, len(h.r.Header))
	}

	// process http 1
	if h.r.ProtoMajor == 1 {
		return h.recvHTTP1(m)
	}

	return h.recvHTTP2(m)
}

func (h *httpTransportSocket) Send(m *Message) error {
	// we need to lock to protect the writer
	h.mtx.RLock()
	defer h.mtx.RUnlock()

	if h.r.ProtoMajor == 1 {
		return h.sendHTTP1(m)
	}

	return h.sendHTTP2(m)
}

func (h *httpTransportSocket) Close() error {
	h.mtx.Lock()
	defer h.mtx.Unlock()

	select {
	case <-h.closed:
		return nil
	default:
		// close the channel
		close(h.closed)

		// Close the buffer
		if err := h.r.Body.Close(); err != nil {
			return err
		}

		// close the connection
		if h.r.ProtoMajor == 1 {
			return h.conn.Close()
		}
	}

	return nil
}

func (h *httpTransportSocket) error(m *Message) error {
	if h.r.ProtoMajor == 1 {
		rsp := &http.Response{
			Header:        make(http.Header),
			Body:          ioutil.NopCloser(bytes.NewReader(m.Body)),
			Status:        "500 Internal Server Error",
			StatusCode:    http.StatusInternalServerError,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: int64(len(m.Body)),
		}

		for k, v := range m.Header {
			rsp.Header.Set(k, v)
		}

		return rsp.Write(h.conn)
	}

	return nil
}

func (h *httpTransportSocket) recvHTTP1(m *Message) error {
	// set timeout if it's greater than 0
	if h.ht.opts.Timeout > time.Duration(0) {
		if err := h.conn.SetDeadline(time.Now().Add(h.ht.opts.Timeout)); err != nil {
			return errors.Wrap(err, "failed to set deadline")
		}
	}

	var r *http.Request

	select {
	// get first request
	case r = <-h.ch:
	// read next request
	default:
		rr, err := http.ReadRequest(h.rw.Reader)
		if err != nil {
			return errors.Wrap(err, "failed to read request")
		}

		r = rr
	}

	// read body
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return errors.Wrap(err, "failed to read body")
	}

	// set body
	if err := r.Body.Close(); err != nil {
		return errors.Wrap(err, "failed to close body")
	}
	m.Body = b

	// set headers
	for k, v := range r.Header {
		if len(v) > 0 {
			m.Header[k] = v[0]
		} else {
			m.Header[k] = ""
		}
	}

	// return early
	return nil
}

func (h *httpTransportSocket) recvHTTP2(m *Message) error {
	// only process if the socket is open
	select {
	case <-h.closed:
		return io.EOF
	default:
		// no op
	}

	// processing http2 request
	// read streaming body

	// set max buffer size
	s := h.ht.opts.BuffSizeH2
	if s == 0 {
		s = DefaultBufSizeH2
	}

	buf := make([]byte, s)

	// read the request body
	n, err := h.buf.Read(buf)
	// not an eof error
	if err != nil {
		return err
	}

	// check if we have data
	if n > 0 {
		m.Body = buf[:n]
	}

	// set headers
	for k, v := range h.r.Header {
		if len(v) > 0 {
			m.Header[k] = v[0]
		} else {
			m.Header[k] = ""
		}
	}

	// set path
	m.Header[":path"] = h.r.URL.Path

	return nil
}

func (h *httpTransportSocket) sendHTTP1(m *Message) error {
	// make copy of header
	hdr := make(http.Header)
	for k, v := range h.r.Header {
		hdr[k] = v
	}

	rsp := &http.Response{
		Header:        hdr,
		Body:          ioutil.NopCloser(bytes.NewReader(m.Body)),
		Status:        "200 OK",
		StatusCode:    http.StatusOK,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: int64(len(m.Body)),
	}

	for k, v := range m.Header {
		rsp.Header.Set(k, v)
	}

	// set timeout if it's greater than 0
	if h.ht.opts.Timeout > time.Duration(0) {
		if err := h.conn.SetDeadline(time.Now().Add(h.ht.opts.Timeout)); err != nil {
			return err
		}
	}

	return rsp.Write(h.conn)
}

func (h *httpTransportSocket) sendHTTP2(m *Message) error {
	// only process if the socket is open
	select {
	case <-h.closed:
		return io.EOF
	default:
		// no op
	}

	// set headers
	for k, v := range m.Header {
		h.w.Header().Set(k, v)
	}

	// write request
	_, err := h.w.Write(m.Body)

	// flush the trailers
	h.w.(http.Flusher).Flush()

	return err
}
