package kafka

import (
	"github.com/IBM/sarama"
	"github.com/realmicro/realmicro/broker"
	"github.com/realmicro/realmicro/logger"
)

// consumerGroupHandler is the implementation of sarama.ConsumerGroupHandler.
type consumerGroupHandler struct {
	handler broker.Handler
	subopts broker.SubscribeOptions
	kopts   broker.Options
	cg      sarama.ConsumerGroup
	sess    sarama.ConsumerGroupSession
}

func (*consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (*consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }
func (h *consumerGroupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var m broker.Message
		p := &publication{m: &m, t: msg.Topic, km: msg, cg: h.cg, sess: sess}
		eh := h.kopts.ErrorHandler

		if err := h.kopts.Codec.Unmarshal(msg.Value, &m); err != nil {
			p.err = err
			p.m.Body = msg.Value
			if eh != nil {
				eh(p)
			} else {
				logger.Errorf("[kafka]: failed to unmarshal: %v", err)
			}
			continue
		}

		if p.m.Body == nil {
			p.m.Body = msg.Value
		}
		// if we don't have headers, create empty map
		if m.Header == nil {
			m.Header = make(map[string]string)
		}
		for _, header := range msg.Headers {
			m.Header[string(header.Key)] = string(header.Value)
		}
		m.Header["Micro-Topic"] = msg.Topic // only for RPC server, it somehow inspect Header for topic
		if _, ok := m.Header["Content-Type"]; !ok {
			m.Header["Content-Type"] = "application/json" // default to json codec
		}

		err := h.handler(p)
		if err == nil && h.subopts.AutoAck {
			logger.Trace("AutoAck: Partition: ", msg.Partition, ", Offset: ", msg.Offset)
			// mark message, not commit
			sess.MarkMessage(msg, "")
		} else if err != nil {
			p.err = err
			if eh != nil {
				eh(p)
			} else {
				logger.Errorf("[kafka]: subscriber error: %v", err)
			}
		}
	}
	return nil
}
