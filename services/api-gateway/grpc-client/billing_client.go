package grpcclient

import (
	"context"

	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/billing"
)

const billingServiceName = "billing-service"

type BillingServiceClient struct {
	Client   pb.BillingServiceClient
	grpcConn *contracts.GRPCClientConn
}

func NewBillingServiceClientWithURL(url string) (*BillingServiceClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: billingServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &BillingServiceClient{
		Client:   pb.NewBillingServiceClient(grpcConn.Conn()),
		grpcConn: grpcConn,
	}, nil
}

func (c *BillingServiceClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *BillingServiceClient) Close() error {
	return c.grpcConn.Close()
}
