package confluentkafka

import "github.com/realmicro/realmicro/broker"

type publication struct {
	t   string
	err error
	m   *broker.Message
}

func (p *publication) Topic() string {
	return p.t
}

func (p *publication) Message() *broker.Message {
	return p.m
}

func (p *publication) Ack() error {
	return nil
}

func (p *publication) Error() error {
	return p.err
}
