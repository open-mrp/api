package grpc

import (
	"context"

	"github.com/open-mrp/api/services/agent-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/notification"
	"github.com/open-mrp/api/shared/rpc"
	"github.com/open-mrp/api/shared/tracing"

	grpclib "google.golang.org/grpc"
)

const notificationServiceName = "notification-service"

var notificationClientTracer = tracing.GetTracer("agent-service.notification_client")

// AgentNotificationClient calls the notification-service EmailBridgeService for the agent's email reply/draft tools.
type AgentNotificationClient struct {
	grpcConn *contracts.GRPCClientConn
	client   pb.EmailBridgeServiceClient
}

func NewAgentNotificationClient(url string) (*AgentNotificationClient, error) {
	grpcConn, err := contracts.NewGRPCClientConn(contracts.GRPCConnTarget{URL: url, Name: notificationServiceName}, nil)
	if err != nil {
		return nil, err
	}
	return &AgentNotificationClient{
		grpcConn: grpcConn,
		client:   pb.NewEmailBridgeServiceClient(grpcConn.Conn()),
	}, nil
}

func (c *AgentNotificationClient) WaitForReady(ctx context.Context) error {
	return c.grpcConn.WaitForReady(ctx)
}

func (c *AgentNotificationClient) Close() error {
	return c.grpcConn.Close()
}

func (c *AgentNotificationClient) SendInboxReply(ctx context.Context, in domain.SendInboxReplyRequest) (string, error) {
	if in.Identity != nil {
		ctx = appctx.WithIdentity(ctx, in.Identity)
	}
	ctx = rpc.PrepareServiceCallCtx(ctx)
	resp, apiErr := rpc.CallRPC(ctx, notificationClientTracer, "notification_client.send_inbox_reply", notificationServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*pb.EmailMessageRef, error) {
			return c.client.SendInboxReply(ctx, &pb.SendInboxReplyRequest{
				ConversationId: in.ConversationID,
				Subject:        in.Subject,
				Body:           in.Body,
				Cc:             in.Cc,
				AgentConfigId:  in.AgentConfigID,
				AgentRunId:     in.AgentRunID,
			}, opts...)
		})
	if apiErr != nil {
		return "", apiErr
	}
	return resp.MessageId, nil
}

func (c *AgentNotificationClient) PostReplyDraft(ctx context.Context, in domain.PostReplyDraftRequest) (string, error) {
	if in.Identity != nil {
		ctx = appctx.WithIdentity(ctx, in.Identity)
	}
	ctx = rpc.PrepareServiceCallCtx(ctx)
	resp, apiErr := rpc.CallRPC(ctx, notificationClientTracer, "notification_client.post_reply_draft", notificationServiceName,
		func(ctx context.Context, opts ...grpclib.CallOption) (*pb.EmailMessageRef, error) {
			return c.client.PostReplyDraft(ctx, &pb.PostReplyDraftRequest{
				ConversationId:        in.ConversationID,
				Body:                  in.Body,
				Subject:               in.Subject,
				AgentConfigId:         in.AgentConfigID,
				AgentRunId:            in.AgentRunID,
				SourceThreadMessageId: in.SourceThreadMessageID,
			}, opts...)
		})
	if apiErr != nil {
		return "", apiErr
	}
	return resp.MessageId, nil
}
