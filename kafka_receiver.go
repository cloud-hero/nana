package nana

import (
	"fmt"
	"reflect"
	"time"

	"github.com/Shopify/sarama"
	"github.com/Sirupsen/logrus"
	"github.com/cenkalti/backoff"
	"github.com/gogo/protobuf/proto"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

const defaultMaxElapsedTime = 15 * time.Minute

// KafkaMessageProcessor is the interface for processing Kafka messages.
type KafkaMessageProcessor interface {
	Message(*sarama.ConsumerMessage, opentracing.Span) error
}

// KafkaReceiver is the Arda implementation for receiving messages from Kafka.
type KafkaReceiver struct {
	log        *logrus.Logger
	tracer     opentracing.Tracer
	processor  KafkaMessageProcessor
	stop       chan struct{}
	subscriber Subscriber

	Name        string
	Backoff     backoff.BackOff
	MessageType interface{}
}

// NewKafkaReceiver configures and returns a KafkaReceiver.
func NewKafkaReceiver(name string, log *logrus.Logger, tracer opentracing.Tracer, subscriber Subscriber, processor KafkaMessageProcessor, messageType interface{}) *KafkaReceiver {
	exponentialBackOff := backoff.NewExponentialBackOff()
	exponentialBackOff.MaxElapsedTime = defaultMaxElapsedTime
	return &KafkaReceiver{
		log:         log,
		tracer:      tracer,
		processor:   processor,
		stop:        make(chan struct{}),
		subscriber:  subscriber,
		Name:        name,
		Backoff:     exponentialBackOff,
		MessageType: messageType,
	}
}

// Start starts an infinite loop waiting for messages from Kafka.
func (r *KafkaReceiver) Start() {
	for {
		select {
		case <-r.stop:
			r.log.Info("Stopping receiver")
			return

		case err := <-r.subscriber.Errors():
			if err != nil {
				r.log.WithFields(logrus.Fields{
					"error": err,
				}).Error("Received error")
			}

		case notification := <-r.subscriber.Notifications():
			if notification != nil {
				r.log.WithFields(logrus.Fields{
					"notification": notification,
				}).Info("Rebalance notification")
			}

		case msg := <-r.subscriber.Messages():
			if msg != nil {

				// Deserialize to get fields for metrics and logging
				messageTypeName := reflect.TypeOf(r.MessageType).Elem().Name()
				logFields := getLogFields(msg, messageTypeName)
				logEntry := r.log.WithFields(logFields)

				decodedMsg := (r.MessageType).(proto.Message)
				err := proto.Unmarshal(msg.Value, decodedMsg)
				if err != nil {
					logEntry.WithField("error", err).Error("Error demarshaling message")
				}

				baggageElem := reflect.ValueOf(decodedMsg).Elem()
				baggageMap := (baggageElem.FieldByName("Baggage").Interface()).(map[string]string)
				mpac := opentracing.TextMapCarrier{}
				mpac[tracer.DefaultParentIDHeader] = baggageMap[tracer.DefaultParentIDHeader]
				mpac[tracer.DefaultTraceIDHeader] = baggageMap[tracer.DefaultTraceIDHeader]
				childCtx, err := r.tracer.Extract(opentracing.TextMap, mpac)
				if err != nil {
					logEntry.WithField("error", err).Error("Error extracting baggage")
				}
				rootSpan := r.tracer.StartSpan(messageTypeName, ext.RPCServerOption(childCtx))
				setTags(msg, messageTypeName, rootSpan)

				operation := func() error {
					err := r.processor.Message(msg, rootSpan)
					if err != nil {
						logEntry.WithField("error", err).Error("Error processing message")
						rootSpan.SetTag("error", fmt.Sprintf("Error processing message: %s", err))
					}
					return err
				}

				err = backoff.Retry(operation, r.Backoff)
				r.Backoff.Reset()
				if err != nil {
					logEntry.WithField("error", err).Error("Too many errors, stopping receiver")
					return
				}

				// Mark message as successfully processed
				r.subscriber.MarkOffset(msg, "")
				logEntry.Info("Processed message")
				rootSpan.Finish()
			}
		}
	}
}

// Stop stops the receiver from receiving any more messages
func (r *KafkaReceiver) Stop() {
	// Send non-blocking to channel so Stop can be called more than once
	select {
	case r.stop <- struct{}{}:
	default:
	}
}

// Close closes the Sarama subscriber.
func (r *KafkaReceiver) Close() error {
	return r.subscriber.Close()
}

func setTags(msg *sarama.ConsumerMessage, messageType string, span opentracing.Span) {
	span.SetTag("topic", msg.Topic)
	span.SetTag("partition", msg.Partition)
	span.SetTag("offset", msg.Offset)
	span.SetTag("messageType", messageType)
}

func getLogFields(msg *sarama.ConsumerMessage, messageType string) logrus.Fields {
	fields := logrus.Fields{
		"topic":        msg.Topic,
		"partition":    msg.Partition,
		"offset":       msg.Offset,
		"messsageType": messageType,
	}

	return fields
}
