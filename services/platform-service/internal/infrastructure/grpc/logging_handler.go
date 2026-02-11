package grpc

import (
	"context"

	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/platform"
	"github.com/augno/api/shared/tracing"

	"google.golang.org/grpc"
)

var loggingGRPCHandlerTracer = tracing.GetTracer("platform-service.logging_grpc_handler")

type loggingHandler struct {
	pb.UnimplementedLoggingServiceServer

	loggingSvc domain.LoggingSvc
}

func NewLoggingHandler(server *grpc.Server, loggingSvc domain.LoggingSvc) *loggingHandler {
	handler := &loggingHandler{
		loggingSvc: loggingSvc,
	}

	pb.RegisterLoggingServiceServer(server, handler)
	return handler
}

func (h *loggingHandler) CreateRequestLog(ctx context.Context, req *pb.CreateRequestLogRequest) (*pb.CreateRequestLogResponse, error) {
	ctx, span := loggingGRPCHandlerTracer.Start(ctx, "grpc_handler.create_request_log")
	defer span.End()

	if req == nil || req.RequestLog == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	rl := &domain.RequestLog{
		ID:                   req.RequestLog.Id,
		Method:               req.RequestLog.Method,
		Host:                 req.RequestLog.Host,
		Path:                 req.RequestLog.Path,
		NormalizedRoute:      req.RequestLog.NormalizedRoute,
		QueryJSON:            req.RequestLog.QueryJson,
		StatusCode:           req.RequestLog.StatusCode,
		LatencyUs:            req.RequestLog.LatencyUs,
		AccountID:            req.RequestLog.AccountId,
		TargetAccountID:      req.RequestLog.TargetAccountId,
		ClientIP:             req.RequestLog.ClientIp,
		ClientIPString:       req.RequestLog.ClientIpString,
		UserAgent:            req.RequestLog.UserAgent,
		Referrer:             req.RequestLog.Referrer,
		ErrorCode:            req.RequestLog.ErrorCode,
		ErrorMessage:         req.RequestLog.ErrorMessage,
		OccurredAt:           req.RequestLog.OccurredAt.AsTime(),
		IdempotencyKeyTypeID: req.RequestLog.IdempotencyKeyId,
		ActorID:              req.RequestLog.ActorId,
		ActorType:            req.RequestLog.ActorType,
		InternalErrorMessage: req.RequestLog.InternalErrorMessage,
		StackTrace:           req.RequestLog.StackTrace,
		IdentityType:         req.RequestLog.IdentityType,
		APIVersion:           req.RequestLog.ApiVersion,
	}

	if req.RequestLog.CreatedAt != nil {
		rl.CreatedAt = req.RequestLog.CreatedAt.AsTime()
	}

	if apiErr := h.loggingSvc.SaveRequestLog(ctx, rl); apiErr != nil {
		tracing.Trace(span, apiErr)
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateRequestLogResponse{
		Success: true,
	}, nil
}
