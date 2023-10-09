package sarama

import (
	"context"

	"github.com/IBM/sarama"
	"github.com/realmicro/realmicro/broker"
)

var (
	DefaultBrokerConfig  = sarama.NewConfig()
	DefaultClusterConfig = sarama.NewConfig()
)

type ProduceErrorHandler func(e *sarama.ProducerError)
type ProduceSuccessHandler func(pm *sarama.ProducerMessage)

type brokerConfigKey struct{}
type clusterConfigKey struct{}

func BrokerConfig(c *sarama.Config) broker.Option {
	return setBrokerOption(brokerConfigKey{}, c)
}

func ClusterConfig(c *sarama.Config) broker.Option {
	return setBrokerOption(clusterConfigKey{}, c)
}

type asyncProduceEnableKey struct{}
type asyncProduceErrorKey struct{}
type asyncProduceSuccessKey struct{}

func AsyncProducerEnable() broker.Option {
	return setBrokerOption(asyncProduceEnableKey{}, true)
}

func AsyncProducerError(errorsHandler ProduceErrorHandler) broker.Option {
	return setBrokerOption(asyncProduceErrorKey{}, errorsHandler)
}

func AsyncProducerSuccess(successesHandler ProduceSuccessHandler) broker.Option {
	return setBrokerOption(asyncProduceSuccessKey{}, successesHandler)
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

type publishMessageKey struct{}

func PublishMessageKey(key string) broker.PublishOption {
	return setPublishOption(publishMessageKey{}, key)
}
