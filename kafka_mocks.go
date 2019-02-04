package nana

import (
	"sync"

	"github.com/Shopify/sarama"
	cluster "github.com/bsm/sarama-cluster"
	opentracing "github.com/opentracing/opentracing-go"
)

var mutex = &sync.Mutex{}

// MockKafkaProcessor is a mock message processor allowing you to verify processed messages.
type MockKafkaProcessor struct {
	MessagesProcessed []*sarama.ConsumerMessage
	MessageAction     func(*sarama.ConsumerMessage, opentracing.Span) error
}

// NewMockKafkaProcessor provides you with a default succeeding mock processor.
func NewMockKafkaProcessor() *MockKafkaProcessor {
	processor := &MockKafkaProcessor{}
	processor.MessageAction = func(msg *sarama.ConsumerMessage, rootSpan opentracing.Span) error {
		mutex.Lock()
		processor.MessagesProcessed = append(processor.MessagesProcessed, msg)
		mutex.Unlock()
		return nil
	}
	return processor
}

// Message performs the configured action against the provided message.
func (p *MockKafkaProcessor) Message(msg *sarama.ConsumerMessage, rootSpan opentracing.Span) error {
	return p.MessageAction(msg, rootSpan)
}

// MockKafkaPublishedMessage is a structure recording the values provided when a message was published.
type MockKafkaPublishedMessage struct {
	Key     string
	Message []byte
	Topic   string
}

// MockKafkaPublisher allows you to mock and verify published messages.
type MockKafkaPublisher struct {
	PublishedMessages []*MockKafkaPublishedMessage
	PublishAction     func(string, []byte, string) error
}

// NewMockKafkaPublisher provides a default, succeeding mock Kafka publisher.
func NewMockKafkaPublisher() *MockKafkaPublisher {
	publisher := &MockKafkaPublisher{}
	publisher.PublishAction = func(key string, message []byte, topic string) error {
		publisher.PublishedMessages = append(publisher.PublishedMessages, &MockKafkaPublishedMessage{
			Key:     key,
			Message: message,
			Topic:   topic,
		})
		return nil
	}
	return publisher
}

// Close does nothing.
func (p *MockKafkaPublisher) Close() error {
	return nil
}

// Publish performs the configured action against the parameters provided.
func (p *MockKafkaPublisher) Publish(key string, message []byte, topic string) error {
	return p.PublishAction(key, message, topic)
}

// MockKafkaSubscriber allows you to mock and verify subscribing to a Kafka topic.
type MockKafkaSubscriber struct {
	MockErrors    []error
	errors        chan error
	notifications chan *cluster.Notification
	messages      chan *sarama.ConsumerMessage
	MarkedOffsets []*sarama.ConsumerMessage
}

// NewMockKafkaSubscriber provides a default, succeeding Kafka subscriber.
func NewMockKafkaSubscriber() *MockKafkaSubscriber {
	return &MockKafkaSubscriber{
		errors:        make(chan error),
		notifications: make(chan *cluster.Notification),
		messages:      make(chan *sarama.ConsumerMessage),
	}
}

// Close cleans up.
func (s *MockKafkaSubscriber) Close() error {
	mutex.Lock()
	close(s.errors)
	close(s.notifications)
	close(s.messages)
	mutex.Unlock()
	return nil
}

// Errors provides a channel exposing the configured errors.
func (s *MockKafkaSubscriber) Errors() <-chan error {
	return s.errors
}

// SendError allows you to simulate Kafka sending an error.
func (s *MockKafkaSubscriber) SendError(err error) {
	s.errors <- err
}

// Notifications allows you to mock re-balance notifications from Kafka.
func (s *MockKafkaSubscriber) Notifications() <-chan *cluster.Notification {
	return s.notifications
}

// SendNotification allows you to simulate a re-balance notification from Kafka.
func (s *MockKafkaSubscriber) SendNotification(notification *cluster.Notification) {
	s.notifications <- notification
}

// Messages is a channel exposing configured messages.
func (s *MockKafkaSubscriber) Messages() <-chan *sarama.ConsumerMessage {
	return s.messages
}

// SendMessage allows you to simulate messages coming from Kafka.
func (s *MockKafkaSubscriber) SendMessage(msg *sarama.ConsumerMessage) {
	s.messages <- msg
}

// MarkOffset records offsets which can then be verified.
func (s *MockKafkaSubscriber) MarkOffset(msg *sarama.ConsumerMessage, metadata string) {
	mutex.Lock()
	s.MarkedOffsets = append(s.MarkedOffsets, msg)
	mutex.Unlock()
}

// GetTopics provides a dummy list of topics.
func (s *MockKafkaSubscriber) GetTopics() []string {
	return []string{"mock-topic"}
}
