package server

import (
	"github.com/realmicro/realmicro/broker"
	"github.com/realmicro/realmicro/transport"
	"github.com/realmicro/realmicro/transport/headers"
)

// event is a broker event we handle on the server transport
type event struct {
	err     error
	message *broker.Message
}

func (e *event) Ack() error {
	// there is no ack support
	return nil
}

func (e *event) Message() *broker.Message {
	return e.message
}

func (e *event) Error() error {
	return e.err
}

func (e *event) Topic() string {
	return e.message.Header[headers.Message]
}

func newEvent(msg transport.Message) *event {
	return &event{
		message: &broker.Message{
			Header: msg.Header,
			Body:   msg.Body,
		},
	}
}
