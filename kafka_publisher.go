package nana

import (
	"crypto/tls"
	"time"

	"github.com/Shopify/sarama"
	"github.com/cenkalti/backoff"

	"github.com/Sirupsen/logrus"
)

// Publisher is the interface for publishing messages.
type Publisher interface {
	Close() error
	Publish(key string, message []byte, topic string) error
}

// KafkaPublisher is the Arda implementation for publishing messages to Kafka.
type KafkaPublisher struct {
	log      *logrus.Logger
	producer sarama.SyncProducer
}

// NewKafkaPublisher configures and provides a KafkaPublisher
func NewKafkaPublisher(log *logrus.Logger, brokers []string, tlsConfig *tls.Config, saslUser string, saslPassword string) (*KafkaPublisher, error) {
	config := producerConfig(tlsConfig, saslUser, saslPassword)

	return NewCustomKafkaPublisher(log, brokers, config)

}

// NewCustomKafkaPublisher tries to create a Sarama producer, retrying for a period of time in case Kafka is starting up.
func NewCustomKafkaPublisher(log *logrus.Logger, brokers []string, config *sarama.Config) (*KafkaPublisher, error) {
	log.Info("Creating Kafka producer")
	var producer sarama.SyncProducer
	var err error

	operation := func() error {
		producer, err = sarama.NewSyncProducer(brokers, config)
		if err != nil {
			log.WithField("error", err).Warn("Error creating producer")
		}
		return err
	}

	exponentialBackoff := backoff.NewExponentialBackOff()
	exponentialBackoff.MaxElapsedTime = KafkaMaxElapsedTimeConnecting
	err = backoff.Retry(operation, exponentialBackoff)

	if err != nil {
		return nil, err
	}

	return NewKafkaPublisherWithProducer(log, producer), nil
}

// NewKafkaPublisherWithProducer allows provides a KafkaPublisher using the provided sarama producer.
func NewKafkaPublisherWithProducer(log *logrus.Logger, producer sarama.SyncProducer) *KafkaPublisher {
	return &KafkaPublisher{
		log:      log,
		producer: producer,
	}
}

// Close closes the Sarama producer.
func (p *KafkaPublisher) Close() error {
	return p.producer.Close()
}

func producerConfig(tlsConfig *tls.Config, saslUser string, saslPassword string) *sarama.Config {
	config := sarama.NewConfig()
	config.Version = KafkaVersion

	config.Net.TLS.Enable = tlsConfig != nil
	config.Net.TLS.Config = tlsConfig
	config.Net.SASL.Enable = len(saslUser) > 0
	config.Net.SASL.User = saslUser
	config.Net.SASL.Password = saslPassword

	config.Producer.MaxMessageBytes = KafkaMaxMessageBytes
	config.Producer.Compression = KafkaMessageCompression
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Partitioner = sarama.NewHashPartitioner
	config.Producer.Retry.Max = 5
	config.Producer.Return.Successes = true

	return config
}

// Publish produces a KafkaMessage and publishes it to the topic
// The key parameter can be a blank string (random partitioning)
// or set to a value so that partitions are set using a hash of the key.
func (p *KafkaPublisher) Publish(key string, message []byte, topic string) error {
	msg := &sarama.ProducerMessage{
		Timestamp: time.Now(),
		Topic:     topic,
		Value:     sarama.StringEncoder(message),
	}

	if len(key) > 0 {
		msg.Key = sarama.StringEncoder(key)
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		return err
	}

	p.log.WithFields(logrus.Fields{
		"partition": partition,
		"offset":    offset,
		"message":   string(message),
	}).Debug("Message published")

	return nil
}
