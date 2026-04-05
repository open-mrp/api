package domain

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/augno/api/services/agent-service/internal/infrastructure/sqlc"
	types "github.com/augno/api/services/auth-service/pkg/types"
)

// HandlerRunContext provides tool handlers with access to repos and accumulated run state.
type HandlerRunContext struct {
	AccountID            string
	RunID                string
	Definition           *sqlc.AgentDefinition
	Config               *sqlc.AgentConfig
	Repos                RepoFactory
	CoreClient           CoreClient
	Identity             *types.Identity
	Actions              []PendingAction
	Artifacts            []PendingArtifact
	Memories             []PendingMemory
	Alerts               []PendingAlert
	RequireReviewBySlug  map[string]bool
	AlwaysAllowedSlugs   map[string]bool // from allowed_tool_slugs, never consumed
	OneTimeApprovedSlugs map[string]bool // from approved_tool_slugs, consumed after execution
}

// ToolHandlerFunc is the signature for a single tool's execution handler.
type ToolHandlerFunc func(ctx context.Context, input json.RawMessage, runCtx *HandlerRunContext) (string, error)

// ToolHandlerRegistry maps tool slugs to their handler functions.
type ToolHandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[string]ToolHandlerFunc
}

func NewToolHandlerRegistry() *ToolHandlerRegistry {
	return &ToolHandlerRegistry{
		handlers: make(map[string]ToolHandlerFunc),
	}
}

func (r *ToolHandlerRegistry) Register(slug string, fn ToolHandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[slug] = fn
}

func (r *ToolHandlerRegistry) Get(slug string) (ToolHandlerFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.handlers[slug]
	return fn, ok
}
