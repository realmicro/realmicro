package confluentkafka

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/realmicro/realmicro/broker"
)

type ProduceErrorHandler func(e *kafka.Error)
type ProduceSuccessHandler func(pm *kafka.Message)

type asyncProduceErrorKey struct{}
type asyncProduceSuccessKey struct{}

func AsyncProducerError(errorsHandler ProduceErrorHandler) broker.Option {
	return setBrokerOption(asyncProduceErrorKey{}, errorsHandler)
}

func AsyncProducerSuccess(successesHandler ProduceSuccessHandler) broker.Option {
	return setBrokerOption(asyncProduceSuccessKey{}, successesHandler)
}

type publishPartitionKey struct{}
type publishMessageKey struct{}

func PublishPartition(partition int32) broker.PublishOption {
	return setPublishOption(publishPartitionKey{}, partition)
}

func PublishMessageKey(key string) broker.PublishOption {
	return setPublishOption(publishMessageKey{}, key)
}

type subscribePartitionKey struct{}
type subscribeSyncKey struct{}

func SubscribePartition(partition int32) broker.SubscribeOption {
	return setSubscribeOption(subscribePartitionKey{}, partition)
}

// SubscribeSync is a subscribe option to enable synchronous message processing
func SubscribeSync() broker.SubscribeOption {
	return setSubscribeOption(subscribeSyncKey{}, true)
}
