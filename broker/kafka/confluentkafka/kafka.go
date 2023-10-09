package confluentkafka

import (
	"strings"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/google/uuid"
	"github.com/realmicro/realmicro/broker"
	"github.com/realmicro/realmicro/logger"
)

type kBroker struct {
	addrs []string

	producer  *kafka.Producer
	consumers []*kafka.Consumer

	connected bool
	scMutex   sync.Mutex
	opts      broker.Options
}

func (k *kBroker) Address() string {
	if len(k.addrs) > 0 {
		return k.addrs[0]
	}
	return "127.0.0.1:9092"
}

func (k *kBroker) Connect() error {
	if k.connected {
		return nil
	}

	k.scMutex.Lock()
	if k.producer != nil {
		k.scMutex.Unlock()
		return nil
	}
	k.scMutex.Unlock()

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": strings.Join(k.addrs, ","),
		// 用户不显示配置时，默认值为1。用户根据自己的业务情况进行设置
		"acks": 1,
		// 请求发生错误时重试次数，建议将该值设置为大于0，失败重试最大程度保证消息不丢失
		"retries": 3,
		// 发送请求失败时到下一次重试请求之间的时间
		"retry.backoff.ms": 100,
		// producer 网络请求的超时时间。
		"socket.timeout.ms": 6000,
		// 设置客户端内部重试间隔。
		"reconnect.backoff.max.ms": 3000,
	})
	if err != nil {
		return err
	}

	var (
		errorsHandler    ProduceErrorHandler
		successesHandler ProduceSuccessHandler
	)
	if h, ok := k.opts.Context.Value(asyncProduceErrorKey{}).(ProduceErrorHandler); ok {
		errorsHandler = h
	}
	if h, ok := k.opts.Context.Value(asyncProduceSuccessKey{}).(ProduceSuccessHandler); ok {
		successesHandler = h
	}

	// When the producer closed, the Events() channel will be closed
	// So the goroutine will auto exit
	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if successesHandler != nil {
					successesHandler(ev)
				} else {
					if ev.TopicPartition.Error != nil {
						logger.Errorf("Delivery failed: %v", ev.TopicPartition.Error)
					} else {
						logger.Tracef("Delivered message to topic %s [%d] at offset %v, key: %s, value: %s",
							*ev.TopicPartition.Topic, ev.TopicPartition.Partition,
							ev.TopicPartition.Offset, string(ev.Key), string(ev.Value))
					}
				}
			case kafka.Error:
				if errorsHandler != nil {
					errorsHandler(&ev)
				} else {
					logger.Errorf("Error: %v", ev)
				}
			}
		}
	}()

	k.scMutex.Lock()
	k.producer = producer
	k.consumers = make([]*kafka.Consumer, 0)
	k.connected = true
	k.scMutex.Unlock()

	return nil
}

func (k *kBroker) Disconnect() error {
	k.scMutex.Lock()
	defer k.scMutex.Unlock()
	for _, consumer := range k.consumers {
		consumer.Close()
	}
	k.consumers = nil
	k.producer.Close()
	k.connected = false
	return nil
}

func (k *kBroker) Init(opts ...broker.Option) error {
	for _, o := range opts {
		o(&k.opts)
	}
	var cAddrs []string
	for _, addr := range k.opts.Addrs {
		if len(addr) == 0 {
			continue
		}
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{"127.0.0.1:9092"}
	}
	k.addrs = cAddrs
	return nil
}

func (k *kBroker) Options() broker.Options {
	return k.opts
}

func (k *kBroker) Publish(topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	b, err := k.opts.Codec.Marshal(msg)
	if err != nil {
		return err
	}

	options := broker.PublishOptions{}
	for _, o := range opts {
		o(&options)
	}

	headers := make([]kafka.Header, 0)
	for hk, hv := range msg.Header {
		headers = append(headers, kafka.Header{
			Key:   hk,
			Value: []byte(hv),
		})
	}

	var partition = kafka.PartitionAny
	var key string
	if options.Context != nil {
		if p, ok := options.Context.Value(publishPartitionKey{}).(int32); ok {
			partition = p
		}
		key, _ = options.Context.Value(publishMessageKey{}).(string)
	}

	m := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: partition,
		},
		Value:   b,
		Headers: headers,
	}
	if len(key) > 0 {
		m.Key = []byte(key)
	}
	if err = k.producer.Produce(m, nil); err != nil {
		return err
	}

	return nil
}

func (k *kBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	opt := broker.SubscribeOptions{
		AutoAck: true,
		Queue:   uuid.New().String(),
	}
	for _, o := range opts {
		o(&opt)
	}

	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": strings.Join(k.addrs, ","),
		"group.id":          opt.Queue,
		"auto.offset.reset": "earliest",

		// 使用 Kafka 消费分组机制时，消费者超时时间。当 Broker 在该时间内没有收到消费者的心跳时，认为该消费者故障失败，Broker
		// 发起重新 Rebalance 过程。目前该值的配置必须在 Broker 配置group.min.session.timeout.ms=6000和group.max.session.timeout.ms=300000 之间
		"session.timeout.ms": 10000,
	})
	if err != nil {
		return nil, err
	}

	k.scMutex.Lock()
	k.consumers = append(k.consumers, consumer)
	k.scMutex.Unlock()

	if err = consumer.SubscribeTopics([]string{topic}, nil); err != nil {
		return nil, err
	}

	s := &subscriber{
		k:        k,
		consumer: consumer,
		opts:     opt,
		t:        topic,
		running:  true,
	}

	go func() {
		for s.running {
			msg, err := consumer.ReadMessage(-1)
			if err == nil {
				// handle
				go func() {
					var m broker.Message
					p := &publication{
						m: &m,
						t: *msg.TopicPartition.Topic,
					}
					eh := k.opts.ErrorHandler
					if err := k.opts.Codec.Unmarshal(msg.Value, &m); err != nil {
						p.err = err
						p.m.Body = msg.Value
						if eh != nil {
							eh(p)
						} else {
							logger.Errorf("[kafka]: failed to unmarshal: %v", err)
						}
						return
					}
					if m.Body == nil {
						m.Body = msg.Value
					}
					if m.Header == nil {
						m.Header = make(map[string]string)
					}
					for i := 0; i < len(msg.Headers); i++ {
						m.Header[msg.Headers[i].Key] = string(msg.Headers[i].Value)
					}
					m.Header["Micro-Topic"] = p.t
					if _, ok := m.Header["Content-Type"]; !ok {
						m.Header["Content-Type"] = "application/json" // default to json codec
					}

					if err := handler(p); err != nil {
						p.err = err
						if eh != nil {
							eh(p)
						} else {
							logger.Errorf("[kafka]: subscriber error: %v", err)
						}
					}
				}()
			} else {
				logger.Errorf("[kafka]: consumer error: %v", err)
			}
		}
	}()

	return s, nil
}

func (k *kBroker) String() string {
	return "kafka"
}

func NewBroker(opts ...broker.Option) broker.Broker {
	options := broker.NewOptions(opts...)

	var cAddrs []string
	for _, addr := range options.Addrs {
		if len(addr) == 0 {
			continue
		}
		cAddrs = append(cAddrs, addr)
	}
	if len(cAddrs) == 0 {
		cAddrs = []string{"127.0.0.1:9092"}
	}

	return &kBroker{
		addrs: cAddrs,
		opts:  options,
	}
}
