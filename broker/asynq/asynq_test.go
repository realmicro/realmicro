package asynq

import (
	"fmt"
	"reflect"
	"sort"
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

	b := NewBroker(
		DB(1),
		Queues(map[string]int{
			"critical": 6,
			"default":  3,
			"low":      1,
		}),
		Service("test"),
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
			fmt.Println("[b1-s0] Received message:", event.Topic(), string(m.Body))
			msgs <- fmt.Sprintf("%s:%s", event.Topic(), string(m.Body))
			return nil
		})

		s1 := subscribe(t, b, "test", func(event broker.Event) error {
			m := event.Message()
			fmt.Println("[b1-s1] Received message:", event.Topic(), string(m.Body))
			msgs <- fmt.Sprintf("%s:%s", event.Topic(), string(m.Body))
			return nil
		}, SubOpr("opr1"))

		s2 := subscribe(t, b, "test", func(event broker.Event) error {
			m := event.Message()
			fmt.Println("[b1-s2] Received message:", event.Topic(), string(m.Body))
			msgs <- fmt.Sprintf("%s:%s", event.Topic(), string(m.Body))
			return nil
		}, SubOpr("opr2"))

		m0 := &broker.Message{
			Body: []byte("empty"),
		}
		publish(t, b, "test0", m0)
		fmt.Println("m0 msg id:", m0.MsgId)

		m1 := &broker.Message{
			Body: []byte("hello"),
		}
		publish(t, b, "test", m1, PubOpr("opr1"))
		fmt.Println("m1 msg id:", m1.MsgId)

		m2 := &broker.Message{
			Body: []byte("world"),
		}
		publish(t, b, "test", m2, PubOpr("opr2"), Queue("critical"), ProcessIn(3*time.Second))
		fmt.Println("m2 msg id:", m2.MsgId)
		if b.Options().Inspector != nil {
			b.Options().Inspector.DeleteTask("critical", m2.MsgId)
		}

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
		//"test:opr2:world",
	}

	fmt.Println(actual)

	// Order is not guaranteed.
	sort.Strings(actual)
	sort.Strings(exp)

	if !reflect.DeepEqual(actual, exp) {
		t.Fatalf("expected %v, got %v", exp, actual)
	}
}
