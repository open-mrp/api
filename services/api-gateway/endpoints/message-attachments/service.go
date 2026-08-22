package attachmentep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/chatmap"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/notification"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

// AttachmentSvc backs the message attachment endpoints via the notification-service
// ChatService gRPC client.
type AttachmentSvc interface {
	CreateAttachmentUploadURL(ctx context.Context, req *CreateAttachmentUploadURLRequest) (*apiresource.AttachmentUploadTarget, *apierror.APIError)
}

type AttachmentSvcConfig struct {
	// ChatClient (required) is the notification-service ChatService gRPC client.
	ChatClient pb.ChatServiceClient
}

type attachmentSvcImpl struct {
	chatClient pb.ChatServiceClient
}

var attachmentSvcTracer = tracing.GetTracer("api-gateway.endpoints.message-attachments.service")

func (c *AttachmentSvcConfig) validate() error {
	if c.ChatClient == nil {
		return fmt.Errorf("message attachments endpoint service: chat client is required")
	}
	return nil
}

func NewAttachmentSvc(config *AttachmentSvcConfig) AttachmentSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &attachmentSvcImpl{chatClient: config.ChatClient}
}

func (s *attachmentSvcImpl) CreateAttachmentUploadURL(ctx context.Context, req *CreateAttachmentUploadURLRequest) (*apiresource.AttachmentUploadTarget, *apierror.APIError) {
	pbReq := &pb.CreateAttachmentUploadURLRequest{
		ConversationId: req.ConversationID,
		Filename:       req.Filename,
		ContentType:    req.ContentType.Ptr(),
	}
	resp, rpcErr := grpcutil.CallRPC(ctx, attachmentSvcTracer, "service.conversations.create_attachment_upload_url", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AttachmentUploadTargetInfo, error) {
			return s.chatClient.CreateAttachmentUploadURL(ctx, pbReq, opts...)
		})
	if rpcErr != nil {
		return nil, rpcErr
	}
	result := &apiresource.AttachmentUploadTarget{
		Object:    constants.ObjectTypeAttachmentUploadTarget,
		UploadURL: resp.UploadUrl,
		S3Key:     resp.S3Key,
		ExpiresAt: chatmap.TsToTime(resp.ExpiresAt),
	}
	if resp.Attachment != nil {
		att := chatmap.AttachmentFromProto(ctx, resp.Attachment)
		resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeAttachmentUploadTarget, resp.S3Key, "attachment", &att)
	}
	return result, nil
}
