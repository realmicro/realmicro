package kafka

import (
	"fmt"
	"github.com/IBM/sarama"
	"testing"
	"time"

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

	kc := sarama.NewConfig()
	kc.Version = sarama.V1_1_1_0
	kc.Producer.RequiredAcks = sarama.WaitForAll

	clusterConfig := sarama.NewConfig()
	clusterConfig.Version = sarama.V1_1_1_0

	b := NewBroker(
		broker.Addrs("127.0.0.1:9092"),
		BrokerConfig(kc),
		ClusterConfig(clusterConfig),
	)

	// Only setting options.
	b.Init()
	if err := b.Connect(); err != nil {
		t.Fatal(err)
	}
	defer b.Disconnect()

	fmt.Println("connect success")

	// Large enough buffer to not block.
	msgs := make(chan string, 10)

	topic := "realmicro-test"
	consumerGroup := "realmicro-test"

	go func() {
		fmt.Println("start subscribe")
		s0 := subscribe(t, b, topic, func(event broker.Event) error {
			m := event.Message()
			fmt.Println("[s0] Received message:", event.Topic(), string(m.Body))
			msgs <- fmt.Sprintf("%s:%s", event.Topic(), string(m.Body))
			return nil
		}, broker.Queue(consumerGroup))

		fmt.Println("start publish")
		m0 := &broker.Message{
			Body: []byte("hello"),
		}
		publish(t, b, topic, m0)
		fmt.Println("m0 msg id:", m0.MsgId)

		m1 := &broker.Message{
			Body: []byte("world"),
		}
		publish(t, b, topic, m1)
		fmt.Println("m1 msg id:", m1.MsgId)

		time.Sleep(10 * time.Second)

		fmt.Println("start unsubscribe")
		unsubscribe(t, s0)

		close(msgs)
	}()

	var actual []string
	for msg := range msgs {
		actual = append(actual, msg)
	}

	fmt.Println(actual)
}
