package grpcclient

import (
	"context"

	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/notification"
)

const notificationServiceName = "notification-service"

// NotificationServiceClient wraps the notification-service gRPC clients (email +
// in-app messaging).
type NotificationServiceClient struct {
	Client            pb.NotificationServiceClient
	MessagingClient   pb.MessagingServiceClient
	ChatClient        pb.ChatServiceClient
	EmailBridgeClient pb.EmailBridgeServiceClient
	grpcConn          *contracts.GRPCClientConn
}

func NewNotificationServiceClientWithURL(url string) (*NotificationServiceClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: notificationServiceName}, nil)
	if err != nil {
		return nil, err
	}

	return &NotificationServiceClient{
		Client:            pb.NewNotificationServiceClient(grpcConn.Conn()),
		MessagingClient:   pb.NewMessagingServiceClient(grpcConn.Conn()),
		ChatClient:        pb.NewChatServiceClient(grpcConn.Conn()),
		EmailBridgeClient: pb.NewEmailBridgeServiceClient(grpcConn.Conn()),
		grpcConn:          grpcConn,
	}, nil
}

func (c *NotificationServiceClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *NotificationServiceClient) Close() error {
	return c.grpcConn.Close()
}
