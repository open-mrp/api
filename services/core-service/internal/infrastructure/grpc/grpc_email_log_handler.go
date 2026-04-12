package grpc

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/contracts"
	pb "github.com/augno/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func emailLogToProto(el *domain.EmailLog) *pb.EmailLogInfo {
	if el == nil {
		return nil
	}

	info := &pb.EmailLogInfo{
		Id:           el.ID,
		HasSent:      el.HasSent,
		Recipients:   el.Recipients,
		Subject:      el.Subject,
		Filename:     el.Filename,
		SesMessageId: el.SESMessageID,
		CreatedAt:    timestamppb.New(el.CreatedAt),
		UpdatedAt:    timestamppb.New(el.UpdatedAt),
	}

	if el.SentBy != nil {
		info.SentBy = &pb.EmailLogActor{
			Id:        el.SentBy.ID,
			ActorType: el.SentBy.ActorType,
			Name:      el.SentBy.Name,
			Handle:    el.SentBy.Handle,
		}
	}

	return info
}

func (h *gRPCHandler) ListEmailLogs(ctx context.Context, req *pb.ListEmailLogsRequest) (*pb.ListEmailLogsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListEmailLogsParams{
		Cursor:   req.Cursor,
		Limit:    req.Limit,
		Query:    req.Query,
		Includes: req.Includes,
	}

	result, apiErr := h.emailLogSvc.ListEmailLogs(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbEmailLogs := make([]*pb.EmailLogInfo, len(result.EmailLogs))
	for i, el := range result.EmailLogs {
		pbEmailLogs[i] = emailLogToProto(el)
	}

	return &pb.ListEmailLogsResponse{
		EmailLogs: pbEmailLogs,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetEmailLog(ctx context.Context, req *pb.GetEmailLogRequest) (*pb.GetEmailLogResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	emailLog, apiErr := h.emailLogSvc.GetEmailLog(ctx, domain.GetEmailLogParams{
		EmailLogID: req.Id,
		Includes:   req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetEmailLogResponse{
		EmailLog: emailLogToProto(emailLog),
	}, nil
}
