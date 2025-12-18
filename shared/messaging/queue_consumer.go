package messaging

import (
	"context"
	"encoding/json"
	"log"

	"github.com/augno/api/shared/contracts"

	amqp "github.com/rabbitmq/amqp091-go"
)

type QueueConsumer struct {
	rb        *RabbitMQ
	connMgr   *ConnectionManager
	queueName string
}

func NewQueueConsumer(rb *RabbitMQ, connMgr *ConnectionManager, queueName string) *QueueConsumer {
	return &QueueConsumer{
		rb:        rb,
		connMgr:   connMgr,
		queueName: queueName,
	}
}

func (qc *QueueConsumer) Start() error {
	return qc.rb.ConsumeMessages(qc.queueName, func(ctx context.Context, msg amqp.Delivery) error {
		var msgBody contracts.AmqpMessage
		if err := json.Unmarshal(msg.Body, &msgBody); err != nil {
			log.Println("Failed to unmarshal message:", err)
			return nil // Return nil to ACK and drop the malformed message
		}

		userID := msgBody.UserID

		var payload any
		if msgBody.Data != nil {
			if err := json.Unmarshal(msgBody.Data, &payload); err != nil {
				log.Println("Failed to unmarshal payload:", err)
				return nil // Return nil to ACK and drop
			}
		}

		clientMsg := contracts.WSMessage{
			Type: msg.RoutingKey,
			Data: payload,
		}

		if err := qc.connMgr.SendMessage(userID, clientMsg); err != nil {
			log.Printf("Failed to send message to user %s: %v", userID, err)
			// Return nil to ACK even if send fails, as we can't do much if user is disconnected
		}

		return nil
	})
}
