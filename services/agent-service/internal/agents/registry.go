package agents

import (
	"sync"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
)

// ToolHandlerRegistry maps tool slugs to their handler functions.
type ToolHandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[string]domain.ToolHandlerFunc
}

func NewToolHandlerRegistry() *ToolHandlerRegistry {
	return &ToolHandlerRegistry{
		handlers: make(map[string]domain.ToolHandlerFunc),
	}
}

func (r *ToolHandlerRegistry) Register(slug string, fn domain.ToolHandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[slug] = fn
}

func (r *ToolHandlerRegistry) Get(slug string) (domain.ToolHandlerFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.handlers[slug]
	return fn, ok
}
