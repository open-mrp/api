package contracts

// WSMessage is a typed WebSocket message with an already-unmarshalled Data payload.
type WSMessage struct {
	// Type identifies the message kind (e.g. "ping", "subscription_update").
	Type string `json:"type"`
	// Data is the message payload, serialized as JSON on the wire.
	Data any `json:"data"`
}

// RunCompleteData is the WSTypeRunComplete payload sent to clients when an agent run finishes. It carries only what the live run view needs to leave its loading state and re-fetch authoritative status. Token usage and model metadata are deliberately excluded — they are never exposed to the frontend.
type RunCompleteData struct {
	AgentRunID string `json:"agent_run_id"`
	AccountID  string `json:"account_id"`
}

// WebSocket message type constants.
const (
	WSTypeSubscribe   = "subscribe"
	WSTypeUnsubscribe = "unsubscribe"
	WSTypePing        = "ping"
	WSTypeRunEvent    = "run_event"
	WSTypeRunComplete = "run_complete"
	WSTypeError       = "error"
	WSTypePong        = "pong"

	// Messaging / notifications — client → server
	WSTypeSubscribeUser           = "subscribe_user"
	WSTypeSubscribeConversation   = "subscribe_conversation"
	WSTypeUnsubscribeConversation = "unsubscribe_conversation"
	WSTypeCatchup                 = "catchup"
	WSTypeMarkRead                = "mark_read"

	// Messaging / notifications — server → client
	WSTypeNotification        = "notification"
	WSTypeMessage             = "message"
	WSTypeConversationUpdated = "conversation_updated"
	WSTypeUnread              = "unread"
	WSTypeTyping              = "typing"
	WSTypeAccountUnreadHint   = "account_unread_hint"
	WSTypeCatchupBatch        = "catchup_batch"
	// WSTypeAgentRunStarted signals on a conversation topic that an agent participant's chat-triggered run has begun; it carries the run id so the client can subscribe to that run's live step stream
	// (run:<id>) and render the agent's interim work inline in the thread.
	WSTypeAgentRunStarted = "agent_run_started"
)
