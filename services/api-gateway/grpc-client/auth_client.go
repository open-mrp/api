package grpcclient

import (
	"context"

	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/auth"
)

const authServiceName = "auth-service"

type AuthServiceClient struct {
	Client   pb.AuthServiceClient
	grpcConn *contracts.GRPCClientConn
}

func NewAuthServiceClientWithURL(url string) (*AuthServiceClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: authServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &AuthServiceClient{
		Client:   pb.NewAuthServiceClient(grpcConn.Conn()),
		grpcConn: grpcConn,
	}, nil
}

func (c *AuthServiceClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *AuthServiceClient) Close() error {
	return c.grpcConn.Close()
}
