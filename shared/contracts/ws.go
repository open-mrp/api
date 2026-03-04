package contracts

// WSMessage is a typed WebSocket message with an already-unmarshalled Data payload.
type WSMessage struct {
	// Type identifies the message kind (e.g. "ping", "subscription_update").
	Type string `json:"type"`
	// Data is the message payload, serialized as JSON on the wire.
	Data any `json:"data"`
}
