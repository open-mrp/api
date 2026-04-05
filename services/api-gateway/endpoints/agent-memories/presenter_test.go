package agentmemoryep

import (
	"testing"

	"github.com/augno/api/services/api-gateway/pkg/resource/resourcetest"
	pb "github.com/augno/api/shared/proto/agent"
)

func TestAgentMemoryPresenter(t *testing.T) {
	t.Parallel()
	now := "2026-01-01T00:00:00Z"

	memory := &pb.AgentMemoryInfo{
		Id:           "agmm_01abc",
		AccountId:    "ac_01abc",
		Category:     "preference",
		Content:      "Customer prefers express shipping.",
		MetadataJson: `{"source":"email"}`,
		EntityType:   "account",
		EntityId:     "ac_01abc",
		Importance:   0.8,
		ExpiresAt:    now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	result := AgentMemoryPresenter(memory)
	resourcetest.ValidateResourceStruct(t, "AgentMemory", result)
}
