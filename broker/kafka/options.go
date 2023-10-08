package kafka

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/realmicro/realmicro/broker"
)

var (
	DefaultBrokerConfig  = sarama.NewConfig()
	DefaultClusterConfig = sarama.NewConfig()
)

type brokerConfigKey struct{}
type clusterConfigKey struct{}

func BrokerConfig(c *sarama.Config) broker.Option {
	return setBrokerOption(brokerConfigKey{}, c)
}

func ClusterConfig(c *sarama.Config) broker.Option {
	return setBrokerOption(clusterConfigKey{}, c)
}

type asyncProduceErrorKey struct{}
type asyncProduceSuccessKey struct{}

func AsyncProducerError(errors chan<- *sarama.ProducerError) broker.Option {
	return setBrokerOption(asyncProduceErrorKey{}, errors)
}

func AsyncProducerMessageSuccess(successes chan<- *sarama.ProducerMessage) broker.Option {
	return setBrokerOption(asyncProduceSuccessKey{}, successes)
}

type subscribeContextKey struct{}

// SubscribeContext set the context for broker.SubscribeOption.
func SubscribeContext(ctx context.Context) broker.SubscribeOption {
	return setSubscribeOption(subscribeContextKey{}, ctx)
}

type subscribeConfigKey struct{}

func SubscribeConfig(c *sarama.Config) broker.SubscribeOption {
	return setSubscribeOption(subscribeConfigKey{}, c)
}
