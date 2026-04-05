package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/augno/api/shared/contracts"
	"github.com/coder/websocket"
)

const (
	sendBufferSize = 256
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingInterval   = 30 * time.Second
)

// Client represents a single WebSocket connection.
type Client struct {
	conn      *websocket.Conn
	hub       *Hub
	send      chan []byte
	accountID string
	runs      map[string]struct{}
}

// NewClient creates a new Client.
func NewClient(conn *websocket.Conn, hub *Hub, accountID string) *Client {
	return &Client{
		conn:      conn,
		hub:       hub,
		send:      make(chan []byte, sendBufferSize),
		accountID: accountID,
		runs:      make(map[string]struct{}),
	}
}

// subscribeMsg is the payload for subscribe/unsubscribe messages.
type subscribeMsg struct {
	RunID        string `json:"run_id"`
	LastSequence *int   `json:"last_sequence,omitempty"`
}

// ReadPump reads messages from the WebSocket connection and handles
// subscribe, unsubscribe, and ping commands.
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
			c.handleSubscribe(msg.Data)
		case contracts.WSTypeUnsubscribe:
			c.handleUnsubscribe(msg.Data)
		case contracts.WSTypePing:
			c.handlePing()
		}
	}
}

func (c *Client) handleSubscribe(data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	var sub subscribeMsg
	if err := json.Unmarshal(raw, &sub); err != nil || sub.RunID == "" {
		return
	}
	c.runs[sub.RunID] = struct{}{}
	c.hub.Subscribe(sub.RunID, c)
	slog.Debug("WS client subscribed", "run_id", sub.RunID, "account_id", c.accountID)
}

func (c *Client) handleUnsubscribe(data any) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	var sub subscribeMsg
	if err := json.Unmarshal(raw, &sub); err != nil || sub.RunID == "" {
		return
	}
	delete(c.runs, sub.RunID)
	c.hub.Unsubscribe(sub.RunID, c)
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
