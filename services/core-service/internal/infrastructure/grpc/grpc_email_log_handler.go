package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	pb "github.com/open-mrp/api/shared/proto/core"
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
		RelatedType:  el.RelatedType,
		RelatedId:    el.RelatedID,
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

// hasSentFilter is the send statuses as a has_sent filter: nil when they name both or neither.
func hasSentFilter(statuses []string) *bool {
	var sent, pending bool
	for _, s := range statuses {
		switch constants.EmailSendStatus(s) {
		case constants.EmailSendStatusSent:
			sent = true
		case constants.EmailSendStatusPending:
			pending = true
		}
	}
	if sent == pending {
		return nil
	}
	return &sent
}

func (h *gRPCHandler) ListEmailLogs(ctx context.Context, req *pb.ListEmailLogsRequest) (*pb.ListEmailLogsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListEmailLogsParams{
		Cursor:        req.Cursor,
		Limit:         req.Limit,
		Query:         req.Query,
		Includes:      req.Includes,
		HasSent:       hasSentFilter(req.SendStatuses),
		CreatedAfter:  timestampPtr(req.CreatedAfter),
		CreatedBefore: timestampPtr(req.CreatedBefore),
		RelatedIDs:    req.RelatedIds,
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
