package agentmemoryep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/agent"
	"github.com/augno/api/shared/tracing"
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
	return &agentMemorySvcImpl{
		agentClient: config.AgentClient,
	}
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
		pbReq.Category = *req.Category
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

	return AgentMemoryListPresenter(ctx, resp), nil
}

func (m *agentMemorySvcImpl) GetMemory(ctx context.Context, req *RetrieveMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
	pbReq := &pb.GetAgentMemoryRequest{
		Id: req.ID,
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, memorySvcTracer, "service.agent_memories.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAgentMemoryResponse, error) {
			return m.agentClient.GetAgentMemory(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	result := AgentMemoryPresenter(resp.Memory)
	return &result, nil
}

func (m *agentMemorySvcImpl) CreateMemory(ctx context.Context, req *CreateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
	pbReq := &pb.CreateAgentMemoryRequest{
		Category:   req.Category,
		Content:    req.Content,
		Importance: req.Importance,
	}
	if req.Metadata != nil {
		pbReq.MetadataJson = string(req.Metadata)
	}
	if req.EntityType != nil {
		pbReq.EntityType = *req.EntityType
	}
	if req.EntityID != nil {
		pbReq.EntityId = *req.EntityID
	}
	if req.ExpiresAt != nil {
		pbReq.ExpiresAt = *req.ExpiresAt
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, memorySvcTracer, "service.agent_memories.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateAgentMemoryResponse, error) {
			return m.agentClient.CreateAgentMemory(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	result := AgentMemoryPresenter(resp.Memory)
	return &result, nil
}

func (m *agentMemorySvcImpl) UpdateMemory(ctx context.Context, req *UpdateMemoryRequest) (*apiresource.AgentMemory, *apierror.APIError) {
	pbReq := &pb.UpdateAgentMemoryRequest{
		Id:         req.ID,
		Category:   req.Category,
		Content:    req.Content,
		Importance: req.Importance,
	}
	if req.Metadata != nil {
		pbReq.MetadataJson = string(req.Metadata)
	}
	if req.EntityType != nil {
		pbReq.EntityType = *req.EntityType
	}
	if req.EntityID != nil {
		pbReq.EntityId = *req.EntityID
	}
	if req.ExpiresAt != nil {
		pbReq.ExpiresAt = *req.ExpiresAt
	}

	resp, rpcErr := grpcutil.CallRPC(ctx, memorySvcTracer, "service.agent_memories.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateAgentMemoryResponse, error) {
			return m.agentClient.UpdateAgentMemory(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	result := AgentMemoryPresenter(resp.Memory)
	return &result, nil
}

func (m *agentMemorySvcImpl) DeleteMemory(ctx context.Context, req *DeleteMemoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteAgentMemoryRequest{
		Id: req.ID,
	}

	_, rpcErr := grpcutil.CallRPC(ctx, memorySvcTracer, "service.agent_memories.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteAgentMemoryResponse, error) {
			return m.agentClient.DeleteAgentMemory(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}

	return &apiresource.EmptyResource{}, nil
}
