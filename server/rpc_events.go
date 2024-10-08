package server

import (
	"context"

	"github.com/realmicro/realmicro/broker"
	raw "github.com/realmicro/realmicro/codec/bytes"
	"github.com/realmicro/realmicro/logger"
	"github.com/realmicro/realmicro/metadata"
	"github.com/realmicro/realmicro/transport/headers"
)

// HandleEvent handles inbound messages to the service directly.
// These events are a result of registering to the topic with the service name.
// TODO: handle requests from an event. We won't send a response.
func (s *rpcServer) HandleEvent(e broker.Event) error {
	// formatting horrible cruft
	msg := e.Message()

	if msg.Header == nil {
		msg.Header = make(map[string]string)
	}

	contentType, ok := msg.Header["Content-Type"]
	if !ok || len(contentType) == 0 {
		msg.Header["Content-Type"] = DefaultContentType
		contentType = DefaultContentType
	}

	cf, err := s.newCodec(contentType)
	if err != nil {
		return err
	}

	header := make(map[string]string, len(msg.Header))
	for k, v := range msg.Header {
		header[k] = v
	}

	// create context
	ctx := metadata.NewContext(context.Background(), header)

	// TODO: inspect message header for Micro-Service & Micro-Topic
	rpcMsg := &rpcMessage{
		topic:       msg.Header[headers.Message],
		contentType: contentType,
		payload:     &raw.Frame{Data: msg.Body},
		codec:       cf,
		header:      msg.Header,
		body:        msg.Body,
	}

	// if the router is present then execute it
	r := Router(s.router)
	if s.opts.Router != nil {
		// create a wrapped function
		handler := s.opts.Router.ProcessMessage

		// execute the wrapper for it
		for i := len(s.opts.SubWrappers); i > 0; i-- {
			handler = s.opts.SubWrappers[i-1](handler)
		}

		// set the router
		r = rpcRouter{m: handler}
	}

	return r.ProcessMessage(ctx, rpcMsg)
}

func (s *rpcServer) NewSubscriber(topic string, sb interface{}, opts ...SubscriberOption) Subscriber {
	return s.router.NewSubscriber(topic, sb, opts...)
}

func (s *rpcServer) Subscribe(sb Subscriber) error {
	s.Lock()
	defer s.Unlock()

	if err := s.router.Subscribe(sb); err != nil {
		return err
	}

	s.subscribers[sb] = nil
	return nil
}

// subscribeServer will subscribe the server to the topic with its own name.
func (s *rpcServer) subscribeServer(config Options) error {
	if s.opts.Router != nil {
		sub, err := s.opts.Broker.Subscribe(config.Name, s.HandleEvent)
		if err != nil {
			return err
		}

		// Save the subscriber
		s.subscriber = sub
	}

	return nil
}

// reSubscribe iterates over subscribers and re-subscribes then.
func (s *rpcServer) reSubscribe(config Options) error {
	for sb := range s.subscribers {
		var opts []broker.SubscribeOption
		if queue := sb.Options().Queue; len(queue) > 0 {
			opts = append(opts, broker.Queue(queue))
		}

		if ctx := sb.Options().Context; ctx != nil {
			opts = append(opts, broker.SubscribeContext(ctx))
		}

		if !sb.Options().AutoAck {
			opts = append(opts, broker.DisableAutoAck())
		}

		if logger.V(logger.InfoLevel, logger.DefaultLogger) {
			logger.Infof("Subscribing to topic: %s", sb.Topic())
		}
		sub, err := config.Broker.Subscribe(sb.Topic(), s.HandleEvent, opts...)
		if err != nil {
			return err
		}

		s.subscribers[sb] = []broker.Subscriber{sub}
		s.router.Subscribe(sb)
	}

	return nil
}
