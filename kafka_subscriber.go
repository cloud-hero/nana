package nana

import (
	"github.com/Shopify/sarama"
	"github.com/Sirupsen/logrus"
	cluster "github.com/bsm/sarama-cluster"
	"github.com/cenkalti/backoff"
)

// Subscriber is the interface for receiving messages.
type Subscriber interface {
	Close() error
	Errors() <-chan error
	Notifications() <-chan *cluster.Notification
	Messages() <-chan *sarama.ConsumerMessage
	MarkOffset(msg *sarama.ConsumerMessage, metadata string)
	GetTopics() []string
}

// KafkaSubscriber is the implementation for receiving messages from Kafka.
type KafkaSubscriber struct {
	consumer *cluster.Consumer
	group    string
	log      *logrus.Logger

	Topics []string
}

// NewKafkaSubscriber configures and returns a KafkaSubscriber.
func NewKafkaSubscriber(log *logrus.Logger, brokers []string, group string, topics []string) (*KafkaSubscriber, error) {
	var err error
	subscriber := &KafkaSubscriber{
		group:  group,
		log:    log,
		Topics: topics,
	}

	config := consumerConfig()
	subscriber.consumer, err = createConsumer(log, brokers, group, topics, config)
	if err != nil {
		return nil, err
	}

	return subscriber, nil
}

// createConsumer tries to create a Sarama consumer, retrying for a period of time in case Kafka is starting up.
func createConsumer(log *logrus.Logger, brokers []string, group string, topics []string, config *cluster.Config) (*cluster.Consumer, error) {
	log.Infof("Creating Kafka consumer for group: %s, topics: %s", group, topics)

	var consumer *cluster.Consumer
	var err error

	operation := func() error {
		consumer, err = cluster.NewConsumer(brokers, group, topics, config)
		if err != nil {
			log.WithField("error", err).Warn("Error creating consumer")
		}
		return err
	}

	exponentialBackoff := backoff.NewExponentialBackOff()
	exponentialBackoff.MaxElapsedTime = KafkaMaxElapsedTimeConnecting
	err = backoff.Retry(operation, exponentialBackoff)
	return consumer, err
}

// Close closes the Sarama consumer.
func (subscriber *KafkaSubscriber) Close() error {
	return subscriber.consumer.Close()
}

// Errors provides any errors returned by Kafka.
func (subscriber *KafkaSubscriber) Errors() <-chan error {
	return subscriber.consumer.Errors()
}

// Notifications provides any re-balance notifications returned by Kafka.
func (subscriber *KafkaSubscriber) Notifications() <-chan *cluster.Notification {
	return subscriber.consumer.Notifications()
}

// Messages provides messages from the Kafka topic.
func (subscriber *KafkaSubscriber) Messages() <-chan *sarama.ConsumerMessage {
	return subscriber.consumer.Messages()
}

// MarkOffset marks the message as processed in Kafka.
func (subscriber *KafkaSubscriber) MarkOffset(msg *sarama.ConsumerMessage, metadata string) {
	subscriber.consumer.MarkOffset(msg, metadata)
}

// GetTopics returns the current topics subscribed to.
func (subscriber *KafkaSubscriber) GetTopics() []string {
	return subscriber.Topics
}

func consumerConfig() *cluster.Config {
	config := cluster.NewConfig()
	config.Version = KafkaVersion

	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Group.Return.Notifications = true

	return config
}
