package grpc

import (
	"context"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/platform"
	"github.com/open-mrp/api/shared/tracing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var accountFollowupGRPCHandlerTracer = tracing.GetTracer("platform-service.account_followup_grpc_handler")

type accountFollowupHandler struct {
	pb.UnimplementedAccountFollowupServiceServer

	svc domain.AccountFollowupSvc
}

func NewAccountFollowupHandler(server *grpc.Server, svc domain.AccountFollowupSvc) *accountFollowupHandler {
	handler := &accountFollowupHandler{svc: svc}
	pb.RegisterAccountFollowupServiceServer(server, handler)
	return handler
}

func (h *accountFollowupHandler) GetAccountFollowupReview(ctx context.Context, req *pb.GetAccountFollowupReviewRequest) (*pb.AccountFollowupReview, error) {
	ctx, span := accountFollowupGRPCHandlerTracer.Start(ctx, "grpc_handler.get_account_followup_review")
	defer span.End()

	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	return reviewResponse(h.svc.GetReview(ctx, req.Token))
}

func (h *accountFollowupHandler) ApproveAccountFollowup(ctx context.Context, req *pb.ApproveAccountFollowupRequest) (*pb.AccountFollowupReview, error) {
	ctx, span := accountFollowupGRPCHandlerTracer.Start(ctx, "grpc_handler.approve_account_followup")
	defer span.End()

	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	return reviewResponse(h.svc.Approve(ctx, req.Token, req.Subject, req.Body))
}

func (h *accountFollowupHandler) SkipAccountFollowup(ctx context.Context, req *pb.SkipAccountFollowupRequest) (*pb.AccountFollowupReview, error) {
	ctx, span := accountFollowupGRPCHandlerTracer.Start(ctx, "grpc_handler.skip_account_followup")
	defer span.End()

	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}
	return reviewResponse(h.svc.Skip(ctx, req.Token))
}

func reviewResponse(r *domain.AccountFollowupReview, apiErr *apierror.APIError) (*pb.AccountFollowupReview, error) {
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	f := r.Followup
	return &pb.AccountFollowupReview{
		Id:              f.ID,
		Status:          string(f.Status),
		RegistrantName:  f.RegistrantName,
		RegistrantEmail: f.RegistrantEmail,
		AccountName:     f.AccountName,
		AccountId:       f.AccountID,
		RegisteredAt:    timestamppb.New(f.RegisteredAt),
		Engagement:      string(f.Engagement),
		InternalSummary: f.InternalSummary,
		Timeline:        r.Timeline,
		DraftSubject:    f.DraftSubject,
		DraftBody:       f.DraftBody,
		FinalSubject:    f.FinalSubject,
		FinalBody:       f.FinalBody,
		ReviewExpiresAt: optionalTimestamp(f.ReviewTokenExpiresAt),
		ReviewedAt:      optionalTimestamp(f.ReviewedAt),
	}, nil
}

func optionalTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
