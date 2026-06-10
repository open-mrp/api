package messaging_test

import (
	"context"

	"github.com/augno/api/shared/messaging"
)

// ExampleNewRabbitMQ shows the minimal configuration for connecting to the
// broker: only URI is required; all other fields receive production defaults.
func ExampleNewRabbitMQ() {
	broker, err := messaging.NewRabbitMQ(context.Background(), &messaging.RabbitMQConfig{
		URI: "amqp://guest:guest@rabbitmq:5672/",
	})
	if err != nil {
		panic(err)
	}
	_ = broker
}
