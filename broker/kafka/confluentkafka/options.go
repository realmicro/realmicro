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
