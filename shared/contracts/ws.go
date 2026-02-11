package contracts

import "encoding/json"

// WSMessage is a typed WebSocket message with an already-unmarshalled Data payload.
type WSMessage struct {
	// Type identifies the message kind (e.g. "ping", "subscription_update").
	Type string `json:"type"`
	// Data is the message payload, serialized as JSON on the wire.
	Data any `json:"data"`
}

// WSDriverMessage is a typed WebSocket message where Data is kept as raw JSON
// so the consumer can defer unmarshalling until the Type is inspected.
type WSDriverMessage struct {
	// Type identifies the message kind.
	Type string `json:"type"`
	// Data is the raw JSON payload, unmarshalled lazily by the consumer.
	Data json.RawMessage `json:"data"`
}
