package tracer

import (
	"reflect"

	"github.com/Shopify/sarama"
	"github.com/gogo/protobuf/proto"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func ExtractBaggage(msg *sarama.ConsumerMessage, messageType interface{}) {
	messageTypeName := reflect.TypeOf(messageType).Elem().Name()

	decodedMsg := (r.MessageType).(proto.Message)

	err := proto.Unmarshal(msg.Value, decodedMsg)

	if err != nil {
		// something here
	}

	baggageElem := reflect.ValueOf(decodedMsg).Elem()
	baggageMap := (baggageElem.FieldByName("Baggage").Interface()).(map[string]string)

	mpac := opentracing.TextMapCarrier{}
	//Datadog specific
	mpac[tracer.DefaultParentIDHeader] = baggageMap[tracer.DefaultParentIDHeader]
	mpac[tracer.DefaultTraceIDHeader] = baggageMap[tracer.DefaultTraceIDHeader]
	childCtx, err := r.tracer.Extract(opentracing.TextMap, mpac)

	if err != nil {
		// error
	}

	rootSpan := r.tracer.StartSpan(messageTypeName, ext.RPCServerOption(childCtx))
}
