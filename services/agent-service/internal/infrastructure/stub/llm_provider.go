package stub

import (
	"context"

	"github.com/augno/api/services/agent-service/internal/llm"
)

// LLMProvider is a no-op LLMProvider implementation for use in test mode.
type LLMProvider struct{}

func (s *LLMProvider) CompleteWithTools(_ context.Context, _ *llm.ToolRequest) (*llm.ToolResponse, error) {
	return &llm.ToolResponse{
		Content:    "stub response",
		StopReason: "end_turn",
	}, nil
}

func (s *LLMProvider) Name() string {
	return "stub"
}
