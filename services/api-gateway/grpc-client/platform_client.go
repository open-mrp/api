package grpcclient

import (
	"context"

	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/platform"
)

const platformServiceName = "platform-service"

type PlatformServiceClient struct {
	Client        pb.IdempotencyServiceClient
	LoggingClient pb.LoggingServiceClient
	AuditClient   pb.AuditServiceClient
	grpcConn      *contracts.GRPCClientConn
}

func NewPlatformServiceClientWithURL(url string) (*PlatformServiceClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: platformServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &PlatformServiceClient{
		Client:        pb.NewIdempotencyServiceClient(grpcConn.Conn()),
		LoggingClient: pb.NewLoggingServiceClient(grpcConn.Conn()),
		AuditClient:   pb.NewAuditServiceClient(grpcConn.Conn()),
		grpcConn:      grpcConn,
	}, nil
}

func (c *PlatformServiceClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *PlatformServiceClient) Close() error {
	return c.grpcConn.Close()
}
