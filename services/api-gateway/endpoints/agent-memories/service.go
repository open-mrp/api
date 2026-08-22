package agentmemoryep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/agent"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

type AgentMemorySvc interface {
	ListMemories(ctx context.Context, req *ListMemoriesRequest) (*apiresource.List[apiresource.AgentMemory], *apierror.APIError)
	GetMemory(ctx context.Context, req *RetrieveMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError)
	CreateMemory(ctx context.Context, req *CreateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError)
	UpdateMemory(ctx context.Context, req *UpdateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError)
	DeleteMemory(ctx context.Context, req *DeleteMemoryRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type AgentMemorySvcConfig struct {
	// AgentClient (required) is the agent-service gRPC client.
	AgentClient pb.AgentServiceClient
}

type agentMemorySvcImpl struct {
	agentClient pb.AgentServiceClient
}

var memorySvcTracer = tracing.GetTracer("api-gateway.endpoints.agent_memories.service")

func (c *AgentMemorySvcConfig) validate() error {
	if c.AgentClient == nil {
		return fmt.Errorf("agent memory endpoint service: agent client is required")
	}
	return nil
}

func NewAgentMemorySvc(config *AgentMemorySvcConfig) AgentMemorySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &agentMemorySvcImpl{agentClient: config.AgentClient}
}

func (m *agentMemorySvcImpl) ListMemories(ctx context.Context, req *ListMemoriesRequest) (*apiresource.List[apiresource.AgentMemory], *apierror.APIError) {
	pbReq := &pb.ListAgentMemoriesRequest{
		Limit:  req.Limit,
		Cursor: req.Cursor,
	}
	if req.Query != nil {
		pbReq.Query = req.Query
	}
	if req.Category != nil {
		pbReq.Category = string(*req.Category)
	}
	if req.EntityType != nil {
		pbReq.EntityType = *req.EntityType
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, memorySvcTracer, "service.agent_memories.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAgentMemoriesResponse, error) {
			return m.agentClient.ListAgentMemories(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	ids := make([]string, len(resp.Memories))
	for i, mem := range resp.Memories {
		ids[i] = mem.Id
	}
	loaded, apiErr := resourceloaders.LoadAgentMemories(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	items := make([]apiresource.AgentMemory, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			items = append(items, *(v.(*apiresource.AgentMemory)))
		}
	}
	pageInfo := apiresource.PageInfo{}
	if resp.PageInfo != nil {
		pageInfo = grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)
	}
	return apiresource.NewList(items, pageInfo), nil
}

func (m *agentMemorySvcImpl) GetMemory(ctx context.Context, req *RetrieveMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
	return loadMemoryByID(ctx, req.ID)
}

func (m *agentMemorySvcImpl) CreateMemory(ctx context.Context, req *CreateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
	pbReq := &pb.CreateAgentMemoryRequest{
		Category: string(req.Category),
		Content:  req.Content,
	}
	if v, ok := req.Importance.Value(); ok {
		pbReq.Importance = v
	}
	if req.Metadata != nil {
		pbReq.MetadataJson = string(req.Metadata)
	}
	if v, ok := req.EntityType.Value(); ok {
		pbReq.EntityType = v
	}
	if v, ok := req.EntityID.Value(); ok {
		pbReq.EntityId = v
	}
	if v, ok := req.ExpiresAt.Value(); ok {
		pbReq.ExpiresAt = v
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, memorySvcTracer, "service.agent_memories.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateAgentMemoryResponse, error) {
			return m.agentClient.CreateAgentMemory(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return loadMemoryByID(ctx, resp.Memory.Id)
}

func (m *agentMemorySvcImpl) UpdateMemory(ctx context.Context, req *UpdateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
	pbReq := &pb.UpdateAgentMemoryRequest{
		Id: req.ID,
	}
	if v, ok := req.Category.Value(); ok {
		pbReq.Category = v.StringPtr()
	}
	if v, ok := req.Content.Value(); ok {
		pbReq.Content = &v
	}
	if v, ok := req.Importance.Value(); ok {
		pbReq.Importance = &v
	}
	if req.Metadata != nil {
		s := string(req.Metadata)
		pbReq.MetadataJson = &s
	}
	if v, ok := req.EntityType.Value(); ok {
		pbReq.EntityType = &v
	}
	if v, ok := req.EntityID.Value(); ok {
		pbReq.EntityId = &v
	}
	// Either entity field sent as null unscopes the memory (both columns cleared together).
	pbReq.ClearEntity = req.EntityType.IsClear() || req.EntityID.IsClear()
	if v, ok := req.ExpiresAt.Value(); ok {
		pbReq.ExpiresAt = &v
	}
	pbReq.ClearExpiresAt = req.ExpiresAt.IsClear()
	resp, rpcErr := grpcutil.CallRPC(ctx, memorySvcTracer, "service.agent_memories.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAgentMemoryResponse, error) {
			return m.agentClient.UpdateAgentMemory(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return loadMemoryByID(ctx, resp.Memory.Id)
}

func (m *agentMemorySvcImpl) DeleteMemory(ctx context.Context, req *DeleteMemoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteAgentMemoryRequest{Id: req.ID}
	_, rpcErr := grpcutil.CallRPC(ctx, memorySvcTracer, "service.agent_memories.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteAgentMemoryResponse, error) {
			return m.agentClient.DeleteAgentMemory(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	return &apiresource.EmptyResource{}, nil
}

// loadMemoryByID wraps the single-ID load pattern used after mutations and
// for the retrieve endpoint.
func loadMemoryByID(ctx context.Context, id string) (*apiresource.AgentMemory, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadAgentMemories(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Agent memory not found.")
	}
	return v.(*apiresource.AgentMemory), nil
}
