package nana

import (
	"errors"
	"testing"

	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestKafkaPublisher(t *testing.T) {
	assert := assert.New(t)

	mockProducer := mocks.NewSyncProducer(t, sarama.NewConfig())
	publisher := NewKafkaPublisherWithProducer(logrus.New(), mockProducer)

	msg := []byte(`{"message":"something"}`)
	mockProducer.ExpectSendMessageWithCheckerFunctionAndSucceed(func(val []byte) error {
		assert.Equal(msg, val)
		return nil
	})

	err := publisher.Publish("key", msg, "my-topic")
	assert.Nil(err)
}

func TestKafkaPublisherSendMessageFail(t *testing.T) {
	assert := assert.New(t)

	mockProducer := mocks.NewSyncProducer(t, sarama.NewConfig())
	publisher := NewKafkaPublisherWithProducer(logrus.New(), mockProducer)

	msg := []byte(`{"message":"something"}`)
	sendErr := errors.New("Failed to send message")
	mockProducer.ExpectSendMessageWithCheckerFunctionAndFail(func(val []byte) error {
		assert.Equal(msg, val)
		return nil
	}, sendErr)

	err := publisher.Publish("key", msg, "my-topic")
	assert.Equal(sendErr, err)
}
