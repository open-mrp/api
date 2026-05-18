package agentmemoryep

import (
	"context"
	"encoding/json"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/agent"
	"github.com/augno/api/shared/timeutil"
)

func AgentMemoryPresenter(m *pb.AgentMemoryInfo) apiresource.AgentMemory {
	if m == nil {
		return apiresource.AgentMemory{}
	}

	memory := apiresource.AgentMemory{
		ID:         m.Id,
		Object:     constants.ObjectTypeAgentMemory,
		Category:   m.Category,
		Content:    m.Content,
		Entity:     entityPresenter(m.EntityType, m.EntityId),
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

func AgentMemoryListPresenter(ctx context.Context, resp *pb.ListAgentMemoriesResponse) *apiresource.List[apiresource.AgentMemory] {
	if resp == nil {
		return apiresource.NewList[apiresource.AgentMemory](nil, apiresource.PageInfo{})
	}

	memories := make([]apiresource.AgentMemory, len(resp.Memories))
	for i, m := range resp.Memories {
		memories[i] = AgentMemoryPresenter(m)
	}

	pageInfo := apiresource.PageInfo{}
	if resp.PageInfo != nil {
		pageInfo = grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)
	}

	return apiresource.NewList(memories, pageInfo)
}

func entityPresenter(entityType, entityID string) *apiresource.Entity {
	if entityType == "" || entityID == "" {
		return nil
	}
	return apiresource.NewEntity(entityID, constants.ObjectType(entityType), nil, nil)
}
