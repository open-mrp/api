package grpcclient

import (
	"context"

	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
)

const coreServiceName = "core-service"

type CoreServiceClient struct {
	Client         pb.CoreServiceClient
	Sales          pb.CoreSalesServiceClient
	Purchase       pb.CorePurchaseServiceClient
	Fulfillment    pb.CoreFulfillmentServiceClient
	Picking        pb.CorePickingServiceClient
	ProductionRun  pb.CoreProductionRunServiceClient
	ProductionStep pb.CoreProductionStepServiceClient
	Receiving      pb.CoreReceivingServiceClient
	Shipping       pb.CoreShippingServiceClient
	ShippingCase   pb.CoreShippingCaseServiceClient
	HubspotSync    pb.CoreHubspotSyncServiceClient
	grpcConn       *contracts.GRPCClientConn
}

func NewCoreServiceClientWithURL(url string) (*CoreServiceClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: coreServiceName}, nil)
	if err != nil {
		return nil, err
	}

	conn := grpcConn.Conn()
	return &CoreServiceClient{
		Client:         pb.NewCoreServiceClient(conn),
		Sales:          pb.NewCoreSalesServiceClient(conn),
		Purchase:       pb.NewCorePurchaseServiceClient(conn),
		Fulfillment:    pb.NewCoreFulfillmentServiceClient(conn),
		Picking:        pb.NewCorePickingServiceClient(conn),
		ProductionRun:  pb.NewCoreProductionRunServiceClient(conn),
		ProductionStep: pb.NewCoreProductionStepServiceClient(conn),
		Receiving:      pb.NewCoreReceivingServiceClient(conn),
		Shipping:       pb.NewCoreShippingServiceClient(conn),
		ShippingCase:   pb.NewCoreShippingCaseServiceClient(conn),
		HubspotSync:    pb.NewCoreHubspotSyncServiceClient(conn),
		grpcConn:       grpcConn,
	}, nil
}

func (c *CoreServiceClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *CoreServiceClient) Close() error {
	return c.grpcConn.Close()
}
