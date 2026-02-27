package grpc

import (
	"context"

	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/notification"
	"github.com/augno/api/shared/rpc"
	"github.com/augno/api/shared/tracing"

	grpclib "google.golang.org/grpc"
)

const notificationServiceName = "notification-service"

var notificationClientTracer = tracing.GetTracer("billing-service.notification_client")

type BillingNotificationClient struct {
	grpcConn *contracts.GRPCClientConn
	client   pb.NotificationServiceClient
}

func NewBillingNotificationClient(url string) (*BillingNotificationClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: notificationServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &BillingNotificationClient{
		grpcConn: grpcConn,
		client:   pb.NewNotificationServiceClient(grpcConn.Conn()),
	}, nil
}

func (c *BillingNotificationClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *BillingNotificationClient) Close() error {
	return c.grpcConn.Close()
}

func (c *BillingNotificationClient) SendEnterpriseRequest(ctx context.Context, accountID, accountName, currentPlanName, requesterName, requesterEmail string) *apierror.APIError {
	ctx = rpc.PrepareRPCCtx(ctx, rpc.WithIdentity(ctx))

	_, apiErr := rpc.CallRPC(ctx, notificationClientTracer, "notification_client.send_enterprise_request", notificationServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*pb.EmailResponse, error) {
			return c.client.SendEnterpriseRequest(ctx, &pb.EnterpriseRequestEmailRequest{
				AccountId:       accountID,
				AccountName:     accountName,
				CurrentPlanName: currentPlanName,
				RequesterName:   requesterName,
				RequesterEmail:  requesterEmail,
			}, opts...)
		})

	return apiErr
}
