package confluent

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/realmicro/realmicro/broker"
)

type subscriber struct {
	k        *kBroker
	t        string
	opts     broker.SubscribeOptions
	running  bool
	consumer *kafka.Consumer
}

func (s *subscriber) Options() broker.SubscribeOptions {
	return s.opts
}

func (s *subscriber) Topic() string {
	return s.t
}

func (s *subscriber) Unsubscribe() error {
	if err := s.consumer.Close(); err != nil {
		return err
	}
	s.running = false

	k := s.k
	k.scMutex.Lock()
	defer k.scMutex.Unlock()

	for i, cg := range k.consumers {
		if cg == s.consumer {
			k.consumers = append(k.consumers[:i], k.consumers[i+1:]...)
			return nil
		}
	}

	return nil
}
