package kafka

import (
	"fmt"
	"testing"

	"github.com/realmicro/realmicro/broker"
	"github.com/realmicro/realmicro/logger"
)

func subscribe(t *testing.T, b broker.Broker, topic string, handle broker.Handler, opts ...broker.SubscribeOption) broker.Subscriber {
	s, err := b.Subscribe(topic, handle, opts...)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func publish(t *testing.T, b broker.Broker, topic string, msg *broker.Message, opts ...broker.PublishOption) {
	if err := b.Publish(topic, msg, opts...); err != nil {
		t.Fatal(err)
	}
}

func unsubscribe(t *testing.T, s broker.Subscriber) {
	if err := s.Unsubscribe(); err != nil {
		t.Fatal(err)
	}
}

func TestBroker(t *testing.T) {
	logger.Init(logger.WithLevel(logger.TraceLevel))

	b := NewBroker(
		broker.Addrs("127.0.0.1:9092"),
	)

	// Only setting options.
	b.Init()
	if err := b.Connect(); err != nil {
		t.Fatal(err)
	}
	defer b.Disconnect()

	// Large enough buffer to not block.
	msgs := make(chan string, 10)

	topic := "realmicro-test"
	consumerGroup := "realmicro-test"

	go func() {
		s0 := subscribe(t, b, topic, func(event broker.Event) error {
			m := event.Message()
			fmt.Println("[s0] Received message:", event.Topic(), string(m.Body))
			msgs <- fmt.Sprintf("%s:%s", event.Topic(), string(m.Body))
			return nil
		}, broker.Queue(consumerGroup))

		m0 := &broker.Message{
			Body: []byte("hello"),
		}
		publish(t, b, topic, m0)
		fmt.Println("m0 msg id:", m0.MsgId)

		m1 := &broker.Message{
			Body: []byte("hello"),
		}
		publish(t, b, topic, m1)
		fmt.Println("m1 msg id:", m1.MsgId)

		unsubscribe(t, s0)

		close(msgs)
	}()

	var actual []string
	for msg := range msgs {
		actual = append(actual, msg)
	}

	fmt.Println(actual)
}
