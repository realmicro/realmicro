package asynq

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/realmicro/realmicro/broker"
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
	b := NewBroker(
		DB(1),
		Queues(map[string]int{
			"critical": 6,
			"default":  3,
			"low":      1,
		}),
	)
	// Only setting options.
	b.Init()
	if err := b.Connect(); err != nil {
		t.Fatal(err)
	}
	defer b.Disconnect()

	// Large enough buffer to not block.
	msgs := make(chan string, 10)

	go func() {
		s0 := subscribe(t, b, "test0", func(event broker.Event) error {
			m := event.Message()
			fmt.Println("[s0] Received message:", event.Topic(), string(m.Body))
			msgs <- fmt.Sprintf("%s:%s", event.Topic(), string(m.Body))
			return nil
		})

		s1 := subscribe(t, b, "test", func(event broker.Event) error {
			m := event.Message()
			fmt.Println("[s1] Received message:", event.Topic(), string(m.Body))
			msgs <- fmt.Sprintf("%s:%s", event.Topic(), string(m.Body))
			return nil
		}, SubOpr("opr1"))

		s2 := subscribe(t, b, "test", func(event broker.Event) error {
			m := event.Message()
			fmt.Println("[s2] Received message:", event.Topic(), string(m.Body))
			msgs <- fmt.Sprintf("%s:%s", event.Topic(), string(m.Body))
			return nil
		}, SubOpr("opr2"))

		publish(t, b, "test0", &broker.Message{
			Body: []byte("empty"),
		})

		publish(t, b, "test", &broker.Message{
			Body: []byte("hello"),
		}, PubOpr("opr1"))

		publish(t, b, "test", &broker.Message{
			Body: []byte("world"),
		}, PubOpr("opr2"), Queue("critical"), ProcessIn(3*time.Second))

		time.Sleep(10 * time.Second)

		unsubscribe(t, s0)
		unsubscribe(t, s1)
		unsubscribe(t, s2)

		close(msgs)
	}()

	var actual []string
	for msg := range msgs {
		actual = append(actual, msg)
	}

	exp := []string{
		"test0:empty",
		"test:opr1:hello",
		"test:opr2:world",
	}

	fmt.Println(actual)

	// Order is not guaranteed.
	sort.Strings(actual)
	sort.Strings(exp)

	if !reflect.DeepEqual(actual, exp) {
		t.Fatalf("expected %v, got %v", exp, actual)
	}
}
