// Package sarama provides a kafka broker using sarama cluster
package sarama

import (
	"context"
	"errors"
	"sync"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/realmicro/realmicro/broker"
	"github.com/realmicro/realmicro/logger"
)

type kBroker struct {
	addrs []string

	c  sarama.Client
	p  sarama.SyncProducer
	ap sarama.AsyncProducer

	cgs []sarama.ConsumerGroup

	connected bool
	scMutex   sync.Mutex
	opts      broker.Options
}

type subscriber struct {
	k    *kBroker
	cg   sarama.ConsumerGroup
	t    string
	opts broker.SubscribeOptions
}

type publication struct {
	t    string
	err  error
	cg   sarama.ConsumerGroup
	km   *sarama.ConsumerMessage
	m    *broker.Message
	sess sarama.ConsumerGroupSession
}

func (p *publication) Topic() string {
	return p.t
}

func (p *publication) Message() *broker.Message {
	return p.m
}

func (p *publication) Ack() error {
	p.sess.MarkMessage(p.km, "")
	return nil
}

func (p *publication) Error() error {
	return p.err
}

func (s *subscriber) Options() broker.SubscribeOptions {
	return s.opts
}

func (s *subscriber) Topic() string {
	return s.t
}

func (s *subscriber) Unsubscribe() error {
	if err := s.cg.Close(); err != nil {
		return err
	}

	k := s.k
	k.scMutex.Lock()
	defer k.scMutex.Unlock()

	for i, cg := range k.cgs {
		if cg == s.cg {
			k.cgs = append(k.cgs[:i], k.cgs[i+1:]...)
			return nil
		}
	}

	return nil
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
	if k.c != nil {
		k.scMutex.Unlock()
		return nil
	}
	k.scMutex.Unlock()

	pconfig := k.getBrokerConfig()
	var (
		asyncProduceEnable, errHandler, successHandler = k.getAsyncProduceHandler()
	)

	pconfig.Producer.Return.Errors = true
	// sync mode
	if !asyncProduceEnable || successHandler != nil {
		pconfig.Producer.Return.Successes = true
	}

	c, err := sarama.NewClient(k.addrs, pconfig)
	if err != nil {
		return err
	}

	var (
		ap sarama.AsyncProducer
		p  sarama.SyncProducer
	)

	if asyncProduceEnable {
		ap, err = sarama.NewAsyncProducerFromClient(c)
		if err != nil {
			return err
		}

		if successHandler != nil {
			go func() {
				for v := range ap.Successes() {
					successHandler(v)
				}
			}()
		}
	} else {
		p, err = sarama.NewSyncProducerFromClient(c)
		if err != nil {
			return err
		}
		go func() {
			for v := range ap.Successes() {
				successHandler(v)
			}
		}()
	}

	// When the ap closed, the Errors() & Successes() channel will be closed
	// So the goroutine will auto exit
	go func() {
		for v := range ap.Errors() {
			if errHandler != nil {
				errHandler(v)
			}
		}
	}()

	k.scMutex.Lock()
	k.c = c
	if p != nil {
		k.p = p
	}
	if ap != nil {
		k.ap = ap
	}
	k.cgs = make([]sarama.ConsumerGroup, 0)
	k.connected = true
	k.scMutex.Unlock()

	return nil
}

func (k *kBroker) Disconnect() error {
	k.scMutex.Lock()
	defer k.scMutex.Unlock()
	for _, consumer := range k.cgs {
		consumer.Close()
	}
	k.cgs = nil
	if k.p != nil {
		k.p.Close()
	}
	if k.ap != nil {
		k.ap.Close()
	}
	if err := k.c.Close(); err != nil {
		return err
	}
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

	var key string
	if options.Context != nil {
		key, _ = options.Context.Value(publishMessageKey{}).(string)
	}

	var produceMsg = &sarama.ProducerMessage{
		Topic:    topic,
		Value:    sarama.ByteEncoder(b),
		Metadata: msg,
	}
	if len(key) > 0 {
		produceMsg.Key = sarama.StringEncoder(key)
	}
	if k.ap != nil {
		k.ap.Input() <- produceMsg
		return nil
	} else if k.p != nil {
		_, _, err = k.p.SendMessage(produceMsg)
		return err
	}

	return errors.New(`no connection resources available`)
}

func (k *kBroker) getSaramaConsumerGroup(groupID string) (sarama.ConsumerGroup, error) {
	config := k.getClusterConfig()
	cg, err := sarama.NewConsumerGroup(k.addrs, groupID, config)
	if err != nil {
		return nil, err
	}
	k.scMutex.Lock()
	defer k.scMutex.Unlock()
	k.cgs = append(k.cgs, cg)
	return cg, nil
}

func (k *kBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	opt := broker.SubscribeOptions{
		AutoAck: true,
		Queue:   uuid.New().String(),
	}
	for _, o := range opts {
		o(&opt)
	}
	// we need to create a new client per consumer
	cg, err := k.getSaramaConsumerGroup(opt.Queue)
	if err != nil {
		return nil, err
	}
	h := &consumerGroupHandler{
		handler: handler,
		subopts: opt,
		kopts:   k.opts,
		cg:      cg,
	}
	ctx := context.Background()
	topics := []string{topic}
	go func() {
		for {
			select {
			case err := <-cg.Errors():
				if err != nil {
					logger.Errorf("consumer error:", err)
				}
			default:
				err := cg.Consume(ctx, topics, h)
				switch err {
				case sarama.ErrClosedConsumerGroup:
					return
				case nil:
					continue
				default:
					logger.Error(err)
				}
			}
		}
	}()
	return &subscriber{k: k, cg: cg, opts: opt, t: topic}, nil
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

func (k *kBroker) getBrokerConfig() *sarama.Config {
	if c, ok := k.opts.Context.Value(brokerConfigKey{}).(*sarama.Config); ok {
		return c
	}
	return DefaultBrokerConfig
}

func (k *kBroker) getAsyncProduceHandler() (bool, ProduceErrorHandler, ProduceSuccessHandler) {
	var (
		asyncProduceEnable bool
		errorsHandler      ProduceErrorHandler
		successesHandler   ProduceSuccessHandler
	)
	if enable, ok := k.opts.Context.Value(asyncProduceEnableKey{}).(bool); ok {
		asyncProduceEnable = enable
	}
	if h, ok := k.opts.Context.Value(asyncProduceErrorKey{}).(ProduceErrorHandler); ok {
		errorsHandler = h
	}
	if h, ok := k.opts.Context.Value(asyncProduceSuccessKey{}).(ProduceSuccessHandler); ok {
		successesHandler = h
	}
	return asyncProduceEnable, errorsHandler, successesHandler
}

func (k *kBroker) getClusterConfig() *sarama.Config {
	if c, ok := k.opts.Context.Value(clusterConfigKey{}).(*sarama.Config); ok {
		return c
	}
	clusterConfig := DefaultClusterConfig
	// the oldest supported version is V1_1_1_0
	if !clusterConfig.Version.IsAtLeast(sarama.V1_1_1_0) {
		clusterConfig.Version = sarama.V1_1_1_0
	}
	clusterConfig.Consumer.Return.Errors = true
	clusterConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	return clusterConfig
}
