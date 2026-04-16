package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/auth"
	"github.com/augno/api/shared/rpc"
	"github.com/augno/api/shared/tracing"

	"google.golang.org/grpc"
)

const authServiceName = "auth-service"

var authClientTracer = tracing.GetTracer("core-service.auth_client")

type CoreAuthClient struct {
	grpcConn *contracts.GRPCClientConn
	client   pb.AuthServiceClient
}

func NewCoreAuthClient(url string) (*CoreAuthClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: authServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &CoreAuthClient{
		grpcConn: grpcConn,
		client:   pb.NewAuthServiceClient(grpcConn.Conn()),
	}, nil
}

func (c *CoreAuthClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *CoreAuthClient) Close() error {
	return c.grpcConn.Close()
}

func (c *CoreAuthClient) GetIncompleteRegistrationSession(ctx context.Context, userID string) (*domain.IncompleteRegistrationSession, *apierror.APIError) {
	ctx = rpc.PrepareServiceCallCtx(ctx)

	resp, apiErr := rpc.CallRPC(ctx, authClientTracer, "auth_client.get_incomplete_registration_session", authServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetIncompleteRegistrationSessionResponse, error) {
			return c.client.GetIncompleteRegistrationSession(ctx, &pb.GetIncompleteRegistrationSessionRequest{
				UserId: userID,
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	if resp.Session == nil {
		return nil, nil
	}

	return &domain.IncompleteRegistrationSession{
		SessionID: resp.Session.SessionId,
		PlanCode:  resp.Session.PlanCode,
		Step:      resp.Session.Step,
		CreatedAt: resp.Session.CreatedAt.AsTime(),
	}, nil
}
