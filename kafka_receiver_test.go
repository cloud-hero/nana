package nana

import (
	"bytes"
	"errors"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/Sirupsen/logrus"
	cluster "github.com/bsm/sarama-cluster"
	"github.com/cenkalti/backoff"
	pb "github.com/cloud-hero/nana/pkg/protobuf/test"
	"github.com/gogo/protobuf/proto"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func setupReceiver(logOutput io.Writer) (*KafkaReceiver, *MockKafkaSubscriber, *MockKafkaProcessor) {
	log := logrus.New()
	log.Formatter = &logrus.TextFormatter{
		DisableColors: true,
	}
	log.Out = logOutput
	msgType := &pb.Test{}
	subscriber := NewMockKafkaSubscriber()
	processor := NewMockKafkaProcessor()

	tracer.Start()
	tracer := opentracing.GlobalTracer()
	receiver := NewKafkaReceiver("testreceiver", log, tracer, subscriber, processor, msgType)
	return receiver, subscriber, processor
}

func encodeMessage() *sarama.ConsumerMessage {
	baggae := make(map[string]string)
	baggae[tracer.DefaultParentIDHeader] = "123"
	baggae[tracer.DefaultTraceIDHeader] = "456"
	msgType := &pb.Test{
		Baggage: baggae,
	}

	data, _ := proto.Marshal(msgType)
	msg := &sarama.ConsumerMessage{}
	msg.Value = data

	return msg
}
func TestKafkaReceiverMessage(t *testing.T) {
	assert := assert.New(t)

	receiver, subscriber, processor := setupReceiver(os.Stdout)
	defer receiver.Close()

	go receiver.Start()

	time.Sleep(100 * time.Millisecond)

	msg := encodeMessage()
	go subscriber.SendMessage(msg)

	time.Sleep(100 * time.Millisecond)

	receiver.Stop()

	assert.Equal(1, len(processor.MessagesProcessed), "Should have processed 1 message")
	assert.Equal(msg, processor.MessagesProcessed[0])

	assert.Equal(1, len(subscriber.MarkedOffsets), "Should have marked offset for 1 message")
	assert.Equal(msg, subscriber.MarkedOffsets[0])
}

func TestStoppingAndStartingKafkaReceiver(t *testing.T) {
	log := logrus.New()
	assert := assert.New(t)

	receiver, subscriber, processor := setupReceiver(os.Stdout)
	defer receiver.Close()

	go receiver.Start()

	time.Sleep(100 * time.Millisecond)

	msg := encodeMessage()
	go func() {
		subscriber.SendMessage(msg)
		subscriber.SendMessage(msg)
		subscriber.SendMessage(msg)
	}()

	time.Sleep(100 * time.Millisecond)

	// Send a bunch of messages after making it inactive
	receiver.Stop()
	time.Sleep(100 * time.Millisecond)

	go func() {
		subscriber.SendMessage(msg)
		subscriber.SendMessage(msg)
		subscriber.SendMessage(msg)
		subscriber.SendMessage(msg)
	}()

	time.Sleep(100 * time.Millisecond)

	// Messages should not be processed and offsets should not be changed
	assert.Equal(3, len(processor.MessagesProcessed), "Should have processed 3 messages")
	assert.Equal(msg, processor.MessagesProcessed[0])

	assert.Equal(3, len(subscriber.MarkedOffsets), "Should have marked offset for 3 messages")
	assert.Equal(msg, subscriber.MarkedOffsets[0])

	// Set it to active mode
	go receiver.Start()
	time.Sleep(100 * time.Millisecond)

	// Set to stop mode
	receiver.Stop()

	// Stop multiple times
	receiver.Stop()
	receiver.Stop()

	// Messages that were sent while it was inactive should be processed now
	log.Infoln("Checking if messages are processed after re-activating")
	assert.Equal(7, len(processor.MessagesProcessed), "Should have processed 7 messages")
	assert.Equal(msg, processor.MessagesProcessed[0])

	assert.Equal(7, len(subscriber.MarkedOffsets), "Should have marked offset for 7 messages")
	assert.Equal(msg, subscriber.MarkedOffsets[0])
}

func TestKafkaReceiverError(t *testing.T) {
	assert := assert.New(t)

	var output string
	logBuffer := bytes.NewBufferString(output)

	receiver, subscriber, _ := setupReceiver(logBuffer)
	defer receiver.Close()

	receiver.Backoff = &backoff.StopBackOff{}

	var mu sync.Mutex

	go func() {
		mu.Lock()
		defer mu.Unlock()
		receiver.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	err := errors.New("An error occurred")
	go subscriber.SendError(err)

	time.Sleep(100 * time.Millisecond)

	receiver.Stop()

	mu.Lock()
	logOutput := logBuffer.String()
	assert.Contains(logOutput, "level=error msg=\"Received error\" error=\"An error occurred\"")
	mu.Unlock()
}

func TestKafkaReceiverNotification(t *testing.T) {
	assert := assert.New(t)

	var output string
	logBuffer := bytes.NewBufferString(output)
	receiver, subscriber, _ := setupReceiver(logBuffer)
	defer receiver.Close()
	receiver.Backoff = &backoff.StopBackOff{}

	var mu sync.Mutex

	go func() {
		mu.Lock()
		defer mu.Unlock()
		receiver.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	notification := &cluster.Notification{}
	go subscriber.SendNotification(notification)

	time.Sleep(100 * time.Millisecond)

	receiver.Stop()

	mu.Lock()
	defer mu.Unlock()

	logOutput := logBuffer.String()
	assert.Contains(logOutput, "level=info msg=\"Rebalance notification\" notification=")
}

func TestKafkaReceiverWhenProcessingMessageFailsReceiverStops(t *testing.T) {
	assert := assert.New(t)

	var output string
	logBuffer := bytes.NewBufferString(output)
	receiver, subscriber, processor := setupReceiver(logBuffer)
	defer receiver.Close()
	receiver.Backoff = &backoff.StopBackOff{}

	var mu sync.Mutex

	processor.MessageAction = func(msg *sarama.ConsumerMessage, span opentracing.Span) error {
		processor.MessagesProcessed = append(processor.MessagesProcessed, msg)
		return errors.New("An error occurred")
	}

	go func() {
		mu.Lock()
		defer mu.Unlock()
		receiver.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	msg := encodeMessage()
	go subscriber.SendMessage(msg)

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(1, len(processor.MessagesProcessed), "Should have processed only 1 message")
	assert.Equal(msg, processor.MessagesProcessed[0])

	assert.Equal(0, len(subscriber.MarkedOffsets), "Should not have marked any offsets")
}

func TestKafkaReceiverWhenProcessingMessageFailsAndRetryBeforeStopReceiver(t *testing.T) {
	assert := assert.New(t)

	receiver, subscriber, processor := setupReceiver(os.Stdout)
	defer receiver.Close()

	// Force to max 3 seconds for the test
	exponentialBackoff := backoff.NewExponentialBackOff()
	exponentialBackoff.MaxElapsedTime = 5 * time.Second
	receiver.Backoff = exponentialBackoff

	receiver.log.Out = os.Stdout

	var mu sync.Mutex

	processor.MessageAction = func(msg *sarama.ConsumerMessage, span opentracing.Span) error {
		mu.Lock()
		defer mu.Unlock()
		processor.MessagesProcessed = append(processor.MessagesProcessed, msg)
		return errors.New("An error occurred")
	}

	go receiver.Start()

	time.Sleep(100 * time.Millisecond)

	msg := encodeMessage()
	go subscriber.SendMessage(msg)

	time.Sleep(1 * time.Second)

	mu.Lock()
	defer mu.Unlock()
	assert.True(len(processor.MessagesProcessed) > 1, "Should have tried to process message several times")
	assert.Equal(msg, processor.MessagesProcessed[0])

	assert.Equal(0, len(subscriber.MarkedOffsets), "Should not have marked any offsets")
}
