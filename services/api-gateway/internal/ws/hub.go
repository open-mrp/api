package ws

import (
	"sync"
)

// Hub manages WebSocket client subscriptions keyed by topic string. Topics namespace the different event streams: "run:<agent_run_id>" (agent run events), "user:<user_id>" (a user's notification bell), and "conv:<conversation_id>" (live chat, future). It fans out incoming events to all clients subscribed to a topic, with tenant isolation via account ID checks.
type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]map[*Client]struct{} // topic → set of clients
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[string]map[*Client]struct{}),
	}
}

// Subscribe registers a client for events on the given topic.
func (h *Hub) Subscribe(topic string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.subscribers[topic] == nil {
		h.subscribers[topic] = make(map[*Client]struct{})
	}
	h.subscribers[topic][client] = struct{}{}
}

// Unsubscribe removes a client from events on the given topic.
func (h *Hub) Unsubscribe(topic string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.subscribers[topic]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.subscribers, topic)
		}
	}
}

// Publish sends an event to all clients subscribed to the given topic. Only clients whose account ID matches the event's account ID receive the message, ensuring tenant isolation.
func (h *Hub) Publish(topic, accountID string, event []byte) {
	h.mu.RLock()
	clients := h.subscribers[topic]
	// Copy the set under read lock to avoid holding it during sends.
	targets := make([]*Client, 0, len(clients))
	for c := range clients {
		if c.accountID == accountID {
			targets = append(targets, c)
		}
	}
	h.mu.RUnlock()

	for _, c := range targets {
		select {
		case c.send <- event:
		default:
			// Client's send buffer is full; drop to avoid blocking.
		}
	}
}

// PublishGlobal sends an event to all clients on a topic WITHOUT the account-isolation check. It is only safe for inherently user-scoped topics (userglobal:<user_id>), where every subscriber is the same user, and is used to carry cross-account unread hints to a user's connections regardless of which account they are currently viewing.
func (h *Hub) PublishGlobal(topic string, event []byte) {
	h.mu.RLock()
	clients := h.subscribers[topic]
	targets := make([]*Client, 0, len(clients))
	for c := range clients {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	for _, c := range targets {
		select {
		case c.send <- event:
		default:
			// Client's send buffer is full; drop to avoid blocking.
		}
	}
}

// RemoveClient unsubscribes a client from all topics it was subscribed to.
func (h *Hub) RemoveClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for topic := range client.topics {
		if clients, ok := h.subscribers[topic]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.subscribers, topic)
			}
		}
	}
}
