package grpc

import (
	"context"

	"github.com/augno/api/services/agent-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/rpc"
	"github.com/augno/api/shared/tracing"

	grpclib "google.golang.org/grpc"
)

const coreServiceName = "core-service"

var coreClientTracer = tracing.GetTracer("agent-service.core_client")

type AgentCoreClient struct {
	grpcConn *contracts.GRPCClientConn
	client   pb.CoreServiceClient
}

func NewAgentCoreClient(url string) (*AgentCoreClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: coreServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &AgentCoreClient{
		grpcConn: grpcConn,
		client:   pb.NewCoreServiceClient(grpcConn.Conn()),
	}, nil
}

func (c *AgentCoreClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *AgentCoreClient) Close() error {
	return c.grpcConn.Close()
}

func prepareCtx(ctx context.Context) context.Context {
	return rpc.PrepareServiceCallCtx(ctx)
}

func (c *AgentCoreClient) GetAccountContext(ctx context.Context, accountID string) (*domain.AccountContext, error) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.get_account_context", coreServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*pb.GetAccountContextResponse, error) {
			return c.client.GetAccountContext(ctx, &pb.GetAccountContextRequest{
				AccountId: accountID,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := &domain.AccountContext{
		IsSandbox:                    resp.IsSandbox,
		PlanCode:                     resp.PlanCode,
		AgentMonthlySpendingCapCents: resp.AgentMonthlySpendingCapCents,
	}
	if resp.OwnerAccountId != nil {
		result.OwnerAccountID = *resp.OwnerAccountId
	}

	return result, nil
}

func (c *AgentCoreClient) GetRolePermissions(ctx context.Context, roleID string) (map[string]bool, error) {
	ctx = prepareCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, coreClientTracer, "core_client.get_role_permissions", coreServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*pb.GetRolePermissionsResponse, error) {
			return c.client.GetRolePermissions(ctx, &pb.GetRolePermissionsRequest{
				RoleId: roleID,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return resp.Permissions, nil
}
