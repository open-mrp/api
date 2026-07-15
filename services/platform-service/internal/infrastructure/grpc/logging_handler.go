package grpc

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/platform"
	"github.com/augno/api/shared/tracing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
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
		PublicEndpoint:       req.RequestLog.PublicEndpoint,
		BodyJSON:             req.RequestLog.BodyJson,
		ResponseJSON:         req.RequestLog.ResponseJson,
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

func (h *loggingHandler) ListRequestLogs(ctx context.Context, req *pb.ListRequestLogsRequest) (*pb.ListRequestLogsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	filter := &domain.ListRequestLogsFilter{
		Query:             req.Query,
		Methods:           req.Methods,
		StatusCodes:       req.StatusCodes,
		StatusCodeClasses: req.StatusCodeClasses,
		ErrorCodes:        req.ErrorCodes,
		ExcludeErrorCodes: req.ExcludeErrorCodes,
		ActorAccountIDs:   req.ActorAccountIds,
		TargetAccountIDs:  req.TargetAccountIds,
		ActorIDs:          req.ActorIds,
		ActorTypes:        req.ActorTypes,
		NormalizedRoutes:  req.NormalizedRoutes,
		Hosts:             req.Hosts,
		MinLatencyUs:      req.MinLatencyUs,
		IdempotencyKey:    req.IdempotencyKey,
		Cursor:            req.Cursor,
		Limit:             req.Limit,
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		filter.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		filter.EndDate = &t
	}

	result, apiErr := h.loggingSvc.ListRequestLogs(ctx, filter, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbLogs := make([]*pb.RequestLogInfo, len(result.RequestLogs))
	for i, rl := range result.RequestLogs {
		pbLogs[i] = requestLogToProto(rl)
	}

	return &pb.ListRequestLogsResponse{
		RequestLogs: pbLogs,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *loggingHandler) GetRequestLog(ctx context.Context, req *pb.GetRequestLogRequest) (*pb.GetRequestLogResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	rl, apiErr := h.loggingSvc.GetRequestLog(ctx, req.Id, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetRequestLogResponse{
		RequestLog: requestLogToProto(rl),
	}, nil
}

func requestLogToProto(rl *domain.RequestLogRead) *pb.RequestLogInfo {
	info := &pb.RequestLogInfo{
		Id:              rl.ID,
		Method:          rl.Method,
		Host:            rl.Host,
		Path:            sanitizeUTF8(rl.Path),
		NormalizedRoute: rl.NormalizedRoute,
		QueryJson:       sanitizeUTF8Ptr(rl.QueryJSON),
		StatusCode:      rl.StatusCode,
		LatencyUs:       rl.LatencyUs,
		ApiVersion:      rl.APIVersion,
		IdentityType:    rl.IdentityType,
		ClientIp:        rl.ClientIP,
		UserAgent:       sanitizeUTF8Ptr(rl.UserAgent),
		Referrer:        sanitizeUTF8Ptr(rl.Referrer),
		ErrorCode:       rl.ErrorCode,
		ErrorMessage:    sanitizeUTF8Ptr(rl.ErrorMessage),
		OccurredAt:      timestamppb.New(rl.OccurredAt),
		CreatedAt:       timestamppb.New(rl.CreatedAt),
		AccountId:       rl.AccountID,
		AccountName:     rl.AccountName,
		IdempotencyKey:  rl.IdempotencyKey,
		BodyJson:        sanitizeUTF8Ptr(rl.BodyJSON),
		ResponseJson:    sanitizeUTF8Ptr(rl.ResponseJSON),
	}

	if rl.AccountCreatedAt != nil {
		info.AccountCreatedAt = timestamppb.New(*rl.AccountCreatedAt)
	}
	if rl.AccountUpdatedAt != nil {
		info.AccountUpdatedAt = timestamppb.New(*rl.AccountUpdatedAt)
	}

	if rl.Actor != nil {
		info.Actor = &pb.RequestLogActor{
			Id:            rl.Actor.ID,
			ActorType:     string(rl.Actor.ActorType),
			Name:          rl.Actor.Name,
			Email:         rl.Actor.Email,
			RedactedValue: rl.Actor.RedactedValue,
			RoleId:        rl.Actor.RoleID,
			RoleName:      rl.Actor.RoleName,
			RoleTypeCode:  rl.Actor.RoleType,
		}
	}

	return info
}

func sanitizeUTF8(s string) string {
	if !strings.ContainsRune(s, utf8.RuneError) {
		return s
	}
	return strings.ToValidUTF8(s, "")
}

func sanitizeUTF8Ptr(s *string) *string {
	if s == nil {
		return nil
	}
	cleaned := sanitizeUTF8(*s)
	return &cleaned
}
