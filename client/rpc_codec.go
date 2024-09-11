package client

import (
	"bytes"
	errs "errors"

	"github.com/realmicro/realmicro/codec"
	raw "github.com/realmicro/realmicro/codec/bytes"
	"github.com/realmicro/realmicro/codec/grpc"
	"github.com/realmicro/realmicro/codec/json"
	"github.com/realmicro/realmicro/codec/jsonrpc"
	"github.com/realmicro/realmicro/codec/proto"
	"github.com/realmicro/realmicro/errors"
	"github.com/realmicro/realmicro/registry"
	"github.com/realmicro/realmicro/transport"
	"github.com/realmicro/realmicro/transport/headers"
)

const (
	lastStreamResponseError = "EOS"
)

// serverError represents an error that has been returned from
// the remote side of the RPC connection.
type serverError string

func (e serverError) Error() string {
	return string(e)
}

// errShutdown holds the specific error for closing/closed connections
var (
	errShutdown = errs.New("connection is shut down")
)

type rpcCodec struct {
	client transport.Client
	codec  codec.Codec

	req *transport.Message
	buf *readWriteCloser

	// signify if its a stream
	stream string
}

type readWriteCloser struct {
	wbuf *bytes.Buffer
	rbuf *bytes.Buffer
}

var (
	DefaultContentType = "application/json"

	DefaultCodecs = map[string]codec.NewCodec{
		"application/grpc":         grpc.NewCodec,
		"application/grpc+json":    grpc.NewCodec,
		"application/grpc+proto":   grpc.NewCodec,
		"application/protobuf":     proto.NewCodec,
		"application/json":         json.NewCodec,
		"application/json-rpc":     jsonrpc.NewCodec,
		"application/octet-stream": raw.NewCodec,
	}
)

func (rwc *readWriteCloser) Read(p []byte) (n int, err error) {
	return rwc.rbuf.Read(p)
}

func (rwc *readWriteCloser) Write(p []byte) (n int, err error) {
	return rwc.wbuf.Write(p)
}

func (rwc *readWriteCloser) Close() error {
	rwc.rbuf.Reset()
	rwc.wbuf.Reset()
	return nil
}

func getHeaders(m *codec.Message) {
	set := func(v, hdr string) string {
		if len(v) > 0 {
			return v
		}
		return m.Header[hdr]
	}

	// check error in header
	m.Error = set(m.Error, headers.Error)

	// check endpoint in header
	m.Endpoint = set(m.Endpoint, headers.Endpoint)

	// check method in header
	m.Method = set(m.Method, headers.Method)

	// set the request id
	m.Id = set(m.Id, headers.ID)
}

func setHeaders(m *codec.Message, stream string) {
	set := func(hdr, v string) {
		if len(v) == 0 {
			return
		}
		m.Header[hdr] = v
	}

	set(headers.ID, m.Id)
	set(headers.Request, m.Target)
	set(headers.Method, m.Method)
	set(headers.Endpoint, m.Endpoint)
	set(headers.Error, m.Error)

	if len(stream) > 0 {
		set(headers.Stream, stream)
	}
}

// setupProtocol sets up the old protocol
func setupProtocol(msg *transport.Message, node *registry.Node) codec.NewCodec {
	protocol := node.Metadata["protocol"]

	// got protocol
	if len(protocol) > 0 {
		return nil
	}

	// processing topic publishing
	if len(msg.Header[headers.Message]) > 0 {
		return nil
	}

	// no protocol use old codecs
	switch msg.Header["Content-Type"] {
	case "application/json":
		msg.Header["Content-Type"] = "application/json-rpc"
	case "application/protobuf":
		msg.Header["Content-Type"] = "application/proto-rpc"
	}

	// now return codec
	return DefaultCodecs[msg.Header["Content-Type"]]
}

func newRpcCodec(req *transport.Message, client transport.Client, c codec.NewCodec, stream string) codec.Codec {
	rwc := &readWriteCloser{
		wbuf: bytes.NewBuffer(nil),
		rbuf: bytes.NewBuffer(nil),
	}
	r := &rpcCodec{
		buf:    rwc,
		client: client,
		codec:  c(rwc),
		req:    req,
		stream: stream,
	}
	return r
}

func (c *rpcCodec) Write(m *codec.Message, body interface{}) error {
	c.buf.wbuf.Reset()

	// create header
	if m.Header == nil {
		m.Header = map[string]string{}
	}

	// copy original header
	for k, v := range c.req.Header {
		m.Header[k] = v
	}

	// set the mucp headers
	setHeaders(m, c.stream)

	// if body is bytes Frame don't encode
	if body != nil {
		if b, ok := body.(*raw.Frame); ok {
			// set body
			m.Body = b.Data
		} else {
			// write to codec
			if err := c.codec.Write(m, body); err != nil {
				return errors.InternalServerError("real.micro.client.codec", err.Error())
			}
			// set body
			m.Body = c.buf.wbuf.Bytes()
		}
	}

	// create new transport message
	msg := transport.Message{
		Header: m.Header,
		Body:   m.Body,
	}

	// send the request
	if err := c.client.Send(&msg); err != nil {
		return errors.InternalServerError("real.micro.client.transport", err.Error())
	}

	return nil
}

func (c *rpcCodec) ReadHeader(m *codec.Message, r codec.MessageType) error {
	var tm transport.Message

	// read message from transport
	if err := c.client.Recv(&tm); err != nil {
		return errors.InternalServerError("real.micro.client.transport", err.Error())
	}

	c.buf.rbuf.Reset()
	c.buf.rbuf.Write(tm.Body)

	// set headers from transport
	m.Header = tm.Header

	// read header
	err := c.codec.ReadHeader(m, r)

	// get headers
	getHeaders(m)

	// return header error
	if err != nil {
		return errors.InternalServerError("real.micro.client.codec", err.Error())
	}

	return nil
}

func (c *rpcCodec) ReadBody(b interface{}) error {
	// read body
	// read raw data
	if v, ok := b.(*raw.Frame); ok {
		v.Data = c.buf.rbuf.Bytes()
		return nil
	}

	if err := c.codec.ReadBody(b); err != nil {
		return errors.InternalServerError("real.micro.client.codec", err.Error())
	}

	return nil
}

func (c *rpcCodec) Close() error {
	if err := c.buf.Close(); err != nil {
		return err
	}

	if err := c.codec.Close(); err != nil {
		return err
	}

	if err := c.client.Close(); err != nil {
		return errors.InternalServerError("real.micro.client.transport", err.Error())
	}

	return nil
}

func (c *rpcCodec) String() string {
	return "rpc"
}
