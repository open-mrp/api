package resourceloaders

import (
	"context"
	"encoding/json"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	agentpb "github.com/augno/api/shared/proto/agent"
	"github.com/augno/api/shared/timeutil"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
)

var agentMemoryLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.agent_memory")

// LoadAgentMemories fetches agent memories by ID via BatchGetAgentMemoriesByIDs (AgentService). Entity is materialized inline from the proto entity_type/entity_id pair — no expandable sub-resources.
func LoadAgentMemories(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, agentMemoryLoaderTracer, "loader.agent_memories.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*agentpb.BatchGetAgentMemoriesByIDsResponse, error) {
			return agentClient.BatchGetAgentMemoriesByIDs(ctx, &agentpb.BatchGetAgentMemoriesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	out := make(map[string]any, len(resp.Memories))
	for _, m := range resp.Memories {
		out[m.Id] = AgentMemoryFromProto(m)
	}
	return out, nil
}

// AgentMemoryFromProto maps the gRPC AgentMemoryInfo to the apiresource shape. Exported so endpoint service methods that already hold a proto response (e.g. Create/Update returning the resource directly) can reuse it.
func AgentMemoryFromProto(m *agentpb.AgentMemoryInfo) *apiresource.AgentMemory {
	memory := &apiresource.AgentMemory{
		ID:         m.Id,
		Object:     constants.ObjectTypeAgentMemory,
		Category:   m.Category,
		Content:    m.Content,
		Entity:     agentMemoryEntityFromProto(m.EntityType, m.EntityId),
		Importance: m.Importance,
		ExpiresAt:  timeutil.TimestampToTimePtr(m.ExpiresAt),
		CreatedAt:  timeutil.TimestampToTime(m.CreatedAt),
		UpdatedAt:  timeutil.TimestampToTime(m.UpdatedAt),
	}
	if m.MetadataJson != "" && m.MetadataJson != "{}" {
		memory.Metadata = json.RawMessage(m.MetadataJson)
	}
	return memory
}

func agentMemoryEntityFromProto(entityType, entityID string) *apiresource.Entity {
	if entityType == "" || entityID == "" {
		return nil
	}
	return apiresource.NewEntity(entityID, constants.ObjectType(entityType), nil, nil)
}
