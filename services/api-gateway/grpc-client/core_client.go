package grpcclient

import (
	"context"

	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
)

const coreServiceName = "core-service"

type CoreServiceClient struct {
	Client   pb.CoreServiceClient
	grpcConn *contracts.GRPCClientConn
}

func NewCoreServiceClient(getenv func(string) string) (*CoreServiceClient, error) {
	return NewCoreServiceClientWithURL(getenv("CORE_SERVICE_URL"))
}

func NewCoreServiceClientWithURL(url string) (*CoreServiceClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: coreServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &CoreServiceClient{
		Client:   pb.NewCoreServiceClient(grpcConn.Conn()),
		grpcConn: grpcConn,
	}, nil
}

func (c *CoreServiceClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *CoreServiceClient) Close() error {
	return c.grpcConn.Close()
}
