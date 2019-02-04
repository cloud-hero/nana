package nana

import (
	"time"

	"github.com/Shopify/sarama"
)

// KafkaVersion is the version of Kafka that Sarama will assume it is running against.
// Sarama defaults to the oldest supported stable version, which does not include
// certain features, so we force it to the latest supported version.
var KafkaVersion = sarama.V2_1_0_0

// KafkaMaxMessageBytes is the maximum message size, set in the Kafka broker settings.
var KafkaMaxMessageBytes = 20971520

// KafkaMessageCompression is the type of compression to use for produced messages.
var KafkaMessageCompression = sarama.CompressionGZIP

// KafkaMaxElapsedTimeConnecting is the maximum amount of time we will retry connecting to Kafka when creating a publisher or subscriber.
const KafkaMaxElapsedTimeConnecting = 1 * time.Minute
