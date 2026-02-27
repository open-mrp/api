package grpcclient

import (
	"context"

	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/platform"
)

const platformServiceName = "platform-service"

type PlatformServiceClient struct {
	Client        pb.IdempotencyServiceClient
	LoggingClient pb.LoggingServiceClient
	grpcConn      *contracts.GRPCClientConn
}

func NewPlatformServiceClient(getenv func(string) string) (*PlatformServiceClient, error) {
	return NewPlatformServiceClientWithURL(getenv("PLATFORM_SERVICE_URL"))
}

func NewPlatformServiceClientWithURL(url string) (*PlatformServiceClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: platformServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &PlatformServiceClient{
		Client:        pb.NewIdempotencyServiceClient(grpcConn.Conn()),
		LoggingClient: pb.NewLoggingServiceClient(grpcConn.Conn()),
		grpcConn:      grpcConn,
	}, nil
}

func (c *PlatformServiceClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *PlatformServiceClient) Close() error {
	return c.grpcConn.Close()
}
