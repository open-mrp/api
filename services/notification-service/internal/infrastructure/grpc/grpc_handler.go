package grpc

import (
	"context"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/notification"

	"google.golang.org/grpc"
)

type gRPCHandler struct {
	pb.UnimplementedNotificationServiceServer
	notificationSvc domain.NotificationSvc
}

// NewGRPCHandler creates and registers the gRPC handler
func NewGRPCHandler(server *grpc.Server, notificationSvc domain.NotificationSvc) *gRPCHandler {
	handler := &gRPCHandler{
		notificationSvc: notificationSvc,
	}
	pb.RegisterNotificationServiceServer(server, handler)
	return handler
}

// SendEmail dispatches an email through the notification service (external email provider).
func (h *gRPCHandler) SendEmail(ctx context.Context, req *pb.EmailRequest) (*pb.EmailResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	_, apiErr := h.notificationSvc.SendEmail(
		ctx,
		domain.EmailSendData{
			To:        req.To,
			Subject:   req.Subject,
			Body:      req.Body,
			SendAs:    req.SendAs,
			AccountID: req.AccountId,
			SentByID:  req.SentById,
		},
	)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.EmailResponse{
		Success: true,
	}, nil
}

// SendEnterpriseRequest emails an enterprise-plan upgrade request (sales notification) through the notification service.
func (h *gRPCHandler) SendEnterpriseRequest(ctx context.Context, req *pb.EnterpriseRequestEmailRequest) (*pb.EmailResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	apiErr := h.notificationSvc.SendEnterpriseRequest(ctx, &domain.EnterpriseRequestData{
		AccountID:       req.AccountId,
		AccountName:     req.AccountName,
		CurrentPlanName: req.CurrentPlanName,
		RequesterName:   req.RequesterName,
		RequesterEmail:  req.RequesterEmail,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.EmailResponse{
		Success: true,
	}, nil
}
