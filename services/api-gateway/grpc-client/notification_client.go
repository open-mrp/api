package grpcclient

import (
	"context"

	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/notification"
)

const notificationServiceName = "notification-service"

type NotificationServiceClient struct {
	Client   pb.NotificationServiceClient
	grpcConn *contracts.GRPCClientConn
}

func NewNotificationServiceClient(getenv func(string) string) (*NotificationServiceClient, error) {
	return NewNotificationServiceClientWithURL(getenv("NOTIFICATION_SERVICE_URL"))
}

func NewNotificationServiceClientWithURL(url string) (*NotificationServiceClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: notificationServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &NotificationServiceClient{
		Client:   pb.NewNotificationServiceClient(grpcConn.Conn()),
		grpcConn: grpcConn,
	}, nil
}

func (c *NotificationServiceClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *NotificationServiceClient) Close() error {
	return c.grpcConn.Close()
}
