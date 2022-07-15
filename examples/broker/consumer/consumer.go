package main

import (
	"fmt"
	"time"

	"github.com/realmicro/realmicro/broker"
	"github.com/realmicro/realmicro/broker/asynq"
	"github.com/realmicro/realmicro/logger"
)

var (
	topic = "realmicro.topic"
)

// Example of a shared subscription which receives a subset of messages
func sharedSub() {
	_, err := broker.Subscribe(topic, func(p broker.Event) error {
		fmt.Println("[sub] received message:", string(p.Message().Body), "header", p.Message().Header)
		return nil
	}, broker.Queue("critical"))
	if err != nil {
		fmt.Println(err)
	}
}

// Example of a subscription which receives all the messages
func sub() {
	_, err := broker.Subscribe(topic, func(p broker.Event) error {
		fmt.Println("[sub] received message:", string(p.Message().Body), "header", p.Message().Header)
		return nil
	})
	if err != nil {
		fmt.Println(err)
	}
}

func main() {
	broker.DefaultBroker = asynq.NewBroker(
		asynq.DB(1),
		asynq.Queues(map[string]int{
			"default": 1,
		}),
		asynq.Service("test"),
	)

	if err := broker.Init(); err != nil {
		logger.Fatalf("Broker Init error: %v", err)
	}
	if err := broker.Connect(); err != nil {
		logger.Fatalf("Broker Connect error: %v", err)
	}

	logger.Info("Start")

	sub()

	<-time.After(time.Minute * 3)
}
