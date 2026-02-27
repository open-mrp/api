package messaging

import (
	"errors"
	"log/slog"
	"net/http"
	"sync"

	"github.com/augno/api/shared/contracts"

	"github.com/gorilla/websocket"
)

var (
	// ErrConnectionNotFound is returned by SendMessage when the target user has no
	// active WebSocket connection in this process. Callers should treat this as a
	// non-fatal condition — the user may have disconnected or may be connected to a
	// different api-gateway replica.
	ErrConnectionNotFound = errors.New("connection not found")
)

// connWrapper guards a single WebSocket connection with a mutex so that concurrent
// goroutines (e.g. multiple message handlers) can safely call WriteJSON without
// interleaving frames. The gorilla/websocket library documents that connections are
// NOT safe for concurrent writes, so every write path must go through the wrapper's
// mutex.
type connWrapper struct {
	conn  *websocket.Conn
	mutex sync.Mutex
}

// ConnectionManager is an in-process registry of active WebSocket connections,
// keyed by user ID. The api-gateway uses it to push real-time notifications to
// connected clients.
//
// Thread safety: all map operations are protected by an RWMutex. Read paths (Get,
// SendMessage) acquire a read lock; write paths (Add, Remove) acquire a full lock.
// Individual writes to a connection are further serialized by connWrapper.mutex.
//
// Limitation: this is a local, single-process store. When the api-gateway scales
// horizontally, each replica only knows about its own connections. A shared backing
// store (e.g. Redis pub/sub) would be needed to fan out messages across replicas.
type ConnectionManager struct {
	connections    map[string]*connWrapper
	mutex          sync.RWMutex
	allowedOrigins []string
}

// NewConnectionManager creates a ConnectionManager with an empty connection map.
// allowedOrigins restricts which origins may connect via WebSocket. If empty,
// all origins are permitted (backward compatible).
func NewConnectionManager(allowedOrigins []string) *ConnectionManager {
	return &ConnectionManager{
		connections:    make(map[string]*connWrapper),
		allowedOrigins: allowedOrigins,
	}
}

// checkOrigin validates the request origin against the allowed origins list.
// If no origins are configured, all origins are accepted for backward compatibility.
func (cm *ConnectionManager) checkOrigin(r *http.Request) bool {
	if len(cm.allowedOrigins) == 0 {
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	for _, allowed := range cm.allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

// Upgrade negotiates the WebSocket handshake on an incoming HTTP request and
// returns the resulting connection. The caller is responsible for adding the
// connection to the manager via Add and for reading from the connection in a
// loop to detect client disconnects.
func (cm *ConnectionManager) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{
		CheckOrigin: cm.checkOrigin,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// Add registers a WebSocket connection for the given user ID, replacing any
// previous connection for the same user (last-write wins). The connection is
// wrapped in a connWrapper for thread-safe writes.
func (cm *ConnectionManager) Add(id string, conn *websocket.Conn) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.connections[id] = &connWrapper{
		conn:  conn,
		mutex: sync.Mutex{},
	}

	slog.Info("Added connection", "user_id", id)
}

// Remove deletes the WebSocket connection for the given user ID. It does NOT close
// the underlying connection — the caller should close it after removing to ensure
// the close frame is sent before the reference is dropped.
func (cm *ConnectionManager) Remove(id string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	delete(cm.connections, id)
}

// Get returns the raw WebSocket connection for the given user ID. The second return
// value is false if the user has no active connection. Note: callers should prefer
// SendMessage for writing, which serializes access through the connWrapper mutex.
func (cm *ConnectionManager) Get(id string) (*websocket.Conn, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	wrapper, exists := cm.connections[id]
	if !exists {
		return nil, false
	}
	return wrapper.conn, true
}

// SendMessage serializes message as JSON and writes it to the WebSocket connection
// for the given user ID. The write is serialized through the connWrapper mutex, so
// it is safe to call concurrently from multiple goroutines. Returns
// ErrConnectionNotFound if the user has no active connection.
func (cm *ConnectionManager) SendMessage(id string, message contracts.WSMessage) error {
	cm.mutex.RLock()
	wrapper, exists := cm.connections[id]
	cm.mutex.RUnlock()

	if !exists {
		return ErrConnectionNotFound
	}

	wrapper.mutex.Lock()
	defer wrapper.mutex.Unlock()

	return wrapper.conn.WriteJSON(message)
}
