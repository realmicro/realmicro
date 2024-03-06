package confluentkafka

import (
	"fmt"
	"os"
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

	addrs := os.Getenv("KAFKA_ADDRESS")
	if len(addrs) == 0 {
		addrs = "127.0.0.1:9092"
	}
	fmt.Println(addrs)

	b := NewBroker(
		broker.Addrs(addrs),
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

		go func() {
			for i := 0; i < 10; i++ {
				m := &broker.Message{
					Body: []byte(fmt.Sprintf("g0#hello#%d", i)),
				}
				publish(t, b, topic, m)
			}
		}()

		m0 := &broker.Message{
			Body: []byte("hello"),
		}
		publish(t, b, topic, m0, PublishMessageKey("hello"))

		m1 := &broker.Message{
			Body: []byte("world"),
		}
		publish(t, b, topic, m1)

		m2 := &broker.Message{
			Body: []byte("partition 0 - message"),
		}
		publish(t, b, topic, m2, PublishPartition(0))

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
	time.Sleep(10 * time.Second)
}

func TestSyncBroker(t *testing.T) {
	logger.Init(logger.WithLevel(logger.TraceLevel))

	addrs := os.Getenv("KAFKA_ADDRESS")
	if len(addrs) == 0 {
		addrs = "127.0.0.1:9092"
	}
	fmt.Println(addrs)

	b := NewBroker(
		broker.Addrs(addrs),
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
			msgs <- fmt.Sprintf("s0:%s:%s", event.Topic(), string(m.Body))
			return nil
		}, broker.Queue(consumerGroup), SubscribeSync())

		s1 := subscribe(t, b, topic, func(event broker.Event) error {
			m := event.Message()
			fmt.Println("[s1] Received message:", event.Topic(), string(m.Body))
			msgs <- fmt.Sprintf("s1:%s:%s", event.Topic(), string(m.Body))
			return nil
		}, broker.Queue(consumerGroup), SubscribeSync())

		fmt.Println("start publish")

		go func() {
			for i := 0; i < 10; i++ {
				m := &broker.Message{
					Body: []byte(fmt.Sprintf("g0#hello#%d", i)),
				}
				publish(t, b, topic, m)
			}
		}()

		go func() {
			for i := 0; i < 10; i++ {
				m := &broker.Message{
					Body: []byte(fmt.Sprintf("g1#hello#%d", i)),
				}
				publish(t, b, topic, m)
				if i%2 == 0 {
					time.Sleep(time.Second)
				}
			}
		}()

		m0 := &broker.Message{
			Body: []byte("hello00#message#key"),
		}
		publish(t, b, topic, m0, PublishMessageKey("mk"))
		m01 := &broker.Message{
			Body: []byte("hello01#message#key"),
		}
		publish(t, b, topic, m01, PublishMessageKey("mk"))
		m02 := &broker.Message{
			Body: []byte("hello02#message#key"),
		}
		publish(t, b, topic, m02, PublishMessageKey("mk"))

		m1 := &broker.Message{
			Body: []byte("world"),
		}
		publish(t, b, topic, m1)

		m2 := &broker.Message{
			Body: []byte("partition#0#message"),
		}
		publish(t, b, topic, m2, PublishPartition(0))

		time.Sleep(10 * time.Second)

		fmt.Println("start unsubscribe")
		unsubscribe(t, s0)
		unsubscribe(t, s1)

		close(msgs)
	}()

	var actual []string
	for msg := range msgs {
		actual = append(actual, msg)
	}

	fmt.Println("msg len:", len(actual), "check == 25")
	fmt.Println(actual)
	time.Sleep(10 * time.Second)
}
