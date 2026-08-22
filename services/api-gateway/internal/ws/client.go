package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/coder/websocket"
	"github.com/open-mrp/api/shared/contracts"
)

const (
	sendBufferSize = 256
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingInterval   = 30 * time.Second
)

// Topic prefixes namespace the Hub's subscription keys.
const (
	topicPrefixRun          = "run:"
	topicPrefixUser         = "user:"
	topicPrefixConversation = "conv:"
	// topicPrefixAccount is the per-account broadcast topic (announcements reach every connected user in the account).
	topicPrefixAccount = "account:"
	// topicPrefixUserGlobal is keyed by user id and is NOT account-isolated; it carries cross-account unread hints to a user's connections regardless of which account they are viewing.
	topicPrefixUserGlobal = "userglobal:"
)

// ParticipantChecker reports whether this connection's user may subscribe to a conversation topic. It is the WS authz gate for conv:<id> (the server fans message events to that topic).
type ParticipantChecker func(ctx context.Context, conversationID string) (bool, error)

// Client represents a single WebSocket connection.
type Client struct {
	conn      *websocket.Conn
	hub       *Hub
	send      chan []byte
	accountID string
	// userID is the connection's user id (us_), the key for its per-user (bell) topic.
	userID           string
	topics           map[string]struct{}
	checkParticipant ParticipantChecker
}

// NewClient creates a new Client.
func NewClient(conn *websocket.Conn, hub *Hub, accountID, userID string, checkParticipant ParticipantChecker) *Client {
	return &Client{
		conn:             conn,
		hub:              hub,
		send:             make(chan []byte, sendBufferSize),
		accountID:        accountID,
		userID:           userID,
		topics:           make(map[string]struct{}),
		checkParticipant: checkParticipant,
	}
}

// subscribe adds the client to a Hub topic and tracks it for cleanup.
func (c *Client) subscribe(topic string) {
	c.topics[topic] = struct{}{}
	c.hub.Subscribe(topic, c)
}

// unsubscribe removes the client from a Hub topic.
func (c *Client) unsubscribe(topic string) {
	delete(c.topics, topic)
	c.hub.Unsubscribe(topic, c)
}

// SubscribeUserTopic subscribes the client to its own notification (bell) topic. Called on connect so a user receives notifications without an explicit subscribe message.
func (c *Client) SubscribeUserTopic() {
	if c.userID == "" {
		return
	}
	c.subscribe(topicPrefixUser + c.userID)
}

// SubscribeAccountTopic subscribes the client to its account's broadcast topic so account-wide announcements arrive live. Called on connect.
func (c *Client) SubscribeAccountTopic() {
	if c.accountID == "" {
		return
	}
	c.subscribe(topicPrefixAccount + c.accountID)
}

// SubscribeUserGlobalTopic subscribes the client to its account-independent user topic, used for cross-account unread hints. Called on connect.
func (c *Client) SubscribeUserGlobalTopic() {
	if c.userID == "" {
		return
	}
	c.subscribe(topicPrefixUserGlobal + c.userID)
}

// subscribeMsg is the payload for subscribe/unsubscribe messages.
type subscribeMsg struct {
	RunID          string `json:"run_id"`
	ConversationID string `json:"conversation_id,omitempty"`
	LastSequence   *int   `json:"last_sequence,omitempty"`
}

// ReadPump reads messages from the WebSocket connection and handles subscribe, unsubscribe, and ping commands.
func (c *Client) ReadPump(ctx context.Context) {
	defer func() {
		c.hub.RemoveClient(c)
		_ = c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return
		}

		var msg contracts.WSMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case contracts.WSTypeSubscribe:
			c.handleSubscribe(ctx, msg.Data)
		case contracts.WSTypeUnsubscribe:
			c.handleUnsubscribe(msg.Data)
		case contracts.WSTypePing:
			c.handlePing()
		}
	}
}

func (c *Client) handleSubscribe(ctx context.Context, data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	var sub subscribeMsg
	if err := json.Unmarshal(raw, &sub); err != nil {
		return
	}

	// Conversation subscriptions are authorized: only active participants may receive a conversation's live message events (the server fans them to conv:<id>).
	if sub.ConversationID != "" {
		if c.checkParticipant == nil {
			return
		}
		ok, err := c.checkParticipant(ctx, sub.ConversationID)
		if err != nil || !ok {
			slog.Debug("WS conversation subscribe denied", "conversation_id", sub.ConversationID, "account_id", c.accountID, "error", err)
			return
		}
		c.subscribe(topicPrefixConversation + sub.ConversationID)
		return
	}

	if sub.RunID != "" {
		c.subscribe(topicPrefixRun + sub.RunID)
		slog.Debug("WS client subscribed", "run_id", sub.RunID, "account_id", c.accountID)
	}
}

func (c *Client) handleUnsubscribe(data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	var sub subscribeMsg
	if err := json.Unmarshal(raw, &sub); err != nil {
		return
	}
	if sub.ConversationID != "" {
		c.unsubscribe(topicPrefixConversation + sub.ConversationID)
		return
	}
	if sub.RunID != "" {
		c.unsubscribe(topicPrefixRun + sub.RunID)
	}
}

func (c *Client) handlePing() {
	pong := contracts.WSMessage{Type: contracts.WSTypePong}
	data, _ := json.Marshal(pong)
	select {
	case c.send <- data:
	default:
	}
}

// WritePump drains the send channel and writes messages to the WebSocket.
func (c *Client) WritePump(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.conn.Write(ctx, websocket.MessageText, msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.Ping(ctx); err != nil {
				return
			}
		}
	}
}
