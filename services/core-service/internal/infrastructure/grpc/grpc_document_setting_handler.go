package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func RegisterDocumentSettingService(server *grpc.Server, documentSettingSvc domain.DocumentSettingSvc) {
	handler.documentSettingSvc = documentSettingSvc
}

func documentSettingToProto(s *domain.DocumentSetting) *pb.DocumentSettingInfo {
	info := &pb.DocumentSettingInfo{
		DocumentType:   string(s.DocumentType),
		ProcessOwner:   s.ProcessOwner,
		DocumentNumber: s.DocumentNumber,
		Revision:       s.Revision,
		FooterText:     s.FooterText,
	}
	if s.ID != "" {
		info.Id = &s.ID
	}
	if s.CreatedAt != nil {
		info.CreatedAt = timestamppb.New(*s.CreatedAt)
	}
	if s.UpdatedAt != nil {
		info.UpdatedAt = timestamppb.New(*s.UpdatedAt)
	}
	return info
}

func (h *gRPCHandler) ListDocumentSettings(ctx context.Context, req *pb.ListDocumentSettingsRequest) (*pb.ListDocumentSettingsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	settings, apiErr := h.documentSettingSvc.ListDocumentSettings(ctx)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	out := make([]*pb.DocumentSettingInfo, len(settings))
	for i, s := range settings {
		out[i] = documentSettingToProto(s)
	}
	return &pb.ListDocumentSettingsResponse{DocumentSettings: out}, nil
}

func (h *gRPCHandler) GetDocumentSetting(ctx context.Context, req *pb.GetDocumentSettingRequest) (*pb.GetDocumentSettingResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	setting, apiErr := h.documentSettingSvc.GetDocumentSetting(ctx, constants.DocumentType(req.DocumentType))
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.GetDocumentSettingResponse{DocumentSetting: documentSettingToProto(setting)}, nil
}

func (h *gRPCHandler) UpdateDocumentSetting(ctx context.Context, req *pb.UpdateDocumentSettingRequest) (*pb.UpdateDocumentSettingResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	setting, apiErr := h.documentSettingSvc.UpdateDocumentSetting(ctx, domain.UpdateDocumentSettingParams{
		DocumentType:   constants.DocumentType(req.DocumentType),
		ProcessOwner:   field.StringClearableFromProto(req.ProcessOwner),
		DocumentNumber: field.StringClearableFromProto(req.DocumentNumber),
		Revision:       field.StringClearableFromProto(req.Revision),
		FooterText:     field.StringClearableFromProto(req.FooterText),
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}
	return &pb.UpdateDocumentSettingResponse{DocumentSetting: documentSettingToProto(setting)}, nil
}
