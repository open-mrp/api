package ws

import (
	"sync"
)

// Hub manages WebSocket client subscriptions, keyed by agent run ID.
// It fans out incoming events to all clients subscribed to a given run,
// with tenant isolation via account ID checks.
type Hub struct {
	mu          sync.RWMutex
	subscribers map[string]map[*Client]struct{} // runID → set of clients
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[string]map[*Client]struct{}),
	}
}

// Subscribe registers a client for events on the given run ID.
func (h *Hub) Subscribe(runID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.subscribers[runID] == nil {
		h.subscribers[runID] = make(map[*Client]struct{})
	}
	h.subscribers[runID][client] = struct{}{}
}

// Unsubscribe removes a client from events on the given run ID.
func (h *Hub) Unsubscribe(runID string, client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.subscribers[runID]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.subscribers, runID)
		}
	}
}

// Publish sends an event to all clients subscribed to the given run ID.
// Only clients whose account ID matches the event's account ID receive
// the message, ensuring tenant isolation.
func (h *Hub) Publish(runID, accountID string, event []byte) {
	h.mu.RLock()
	clients := h.subscribers[runID]
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

// RemoveClient unsubscribes a client from all runs it was subscribed to.
func (h *Hub) RemoveClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for runID := range client.runs {
		if clients, ok := h.subscribers[runID]; ok {
			delete(clients, client)
			if len(clients) == 0 {
				delete(h.subscribers, runID)
			}
		}
	}
}
