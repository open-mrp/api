package grpc

import (
	"context"
	"fmt"

	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/platform"
)

const platformServiceName = "platform-service"

type PlatformServiceClient struct {
	Client   pb.IdempotencyServiceClient
	grpcConn *contracts.GRPCClientConn
}

func NewPlatformServiceClient(url string) (*PlatformServiceClient, error) {
	if url == "" {
		return nil, fmt.Errorf("platform service URL is required")
	}

	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: platformServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &PlatformServiceClient{
		Client:   pb.NewIdempotencyServiceClient(grpcConn.Conn()),
		grpcConn: grpcConn,
	}, nil
}

func (c *PlatformServiceClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *PlatformServiceClient) Close() error {
	return c.grpcConn.Close()
}
