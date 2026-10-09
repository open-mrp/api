package documentsettingep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

type DocumentSettingSvc interface {
	ListDocumentSettings(ctx context.Context, req *ListDocumentSettingsRequest) (*apiresource.List[apiresource.DocumentSetting], *apierror.APIError)
	GetDocumentSetting(ctx context.Context, req *RetrieveDocumentSettingRequest) (*apiresource.DocumentSetting, *apierror.APIError)
	UpdateDocumentSetting(ctx context.Context, req *UpdateDocumentSettingRequest) (*apiresource.DocumentSetting, *apierror.APIError)
}

type DocumentSettingSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
	CoreClient pb.CoreServiceClient
}

type documentSettingSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var documentSettingSvcTracer = tracing.GetTracer("api-gateway.endpoints.document_settings.service")

func (c *DocumentSettingSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("document setting endpoint service: core client is required")
	}
	return nil
}

func NewDocumentSettingSvc(config *DocumentSettingSvcConfig) DocumentSettingSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &documentSettingSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *documentSettingSvcImpl) ListDocumentSettings(ctx context.Context, _ *ListDocumentSettingsRequest) (*apiresource.List[apiresource.DocumentSetting], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, documentSettingSvcTracer, "service.document_settings.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListDocumentSettingsResponse, error) {
			return m.coreClient.ListDocumentSettings(ctx, &pb.ListDocumentSettingsRequest{}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	settings := make([]apiresource.DocumentSetting, len(resp.DocumentSettings))
	for i, s := range resp.DocumentSettings {
		settings[i] = documentSettingFromProto(s)
	}
	return apiresource.NewList(settings, apiresource.PageInfo{}), nil
}

func (m *documentSettingSvcImpl) GetDocumentSetting(ctx context.Context, req *RetrieveDocumentSettingRequest) (*apiresource.DocumentSetting, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, documentSettingSvcTracer, "service.document_settings.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetDocumentSettingResponse, error) {
			return m.coreClient.GetDocumentSetting(ctx, &pb.GetDocumentSettingRequest{DocumentType: string(req.DocumentType)}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := documentSettingFromProto(resp.DocumentSetting)
	return &result, nil
}

func (m *documentSettingSvcImpl) UpdateDocumentSetting(ctx context.Context, req *UpdateDocumentSettingRequest) (*apiresource.DocumentSetting, *apierror.APIError) {
	pbReq := &pb.UpdateDocumentSettingRequest{
		DocumentType: string(req.DocumentType),
		FooterText:   field.StringClearableToProto(req.FooterText),
	}
	if control, ok := req.DocumentControl.Value(); ok {
		pbReq.ProcessOwner = field.StringClearableToProto(control.ProcessOwner)
		pbReq.DocumentNumber = field.StringClearableToProto(control.DocumentNumber)
		pbReq.Revision = field.StringClearableToProto(control.Revision)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, documentSettingSvcTracer, "service.document_settings.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateDocumentSettingResponse, error) {
			return m.coreClient.UpdateDocumentSetting(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := documentSettingFromProto(resp.DocumentSetting)
	return &result, nil
}

func documentSettingFromProto(s *pb.DocumentSettingInfo) apiresource.DocumentSetting {
	if s == nil {
		return apiresource.DocumentSetting{}
	}

	result := apiresource.DocumentSetting{
		ID:           s.Id,
		Object:       constants.ObjectTypeDocumentSetting,
		DocumentType: constants.DocumentType(s.DocumentType),
		DocumentControl: apiresource.DocumentControl{
			Object:         constants.ObjectTypeDocumentControl,
			ProcessOwner:   s.ProcessOwner,
			DocumentNumber: s.DocumentNumber,
			Revision:       s.Revision,
		},
		FooterText: s.FooterText,
	}
	if s.CreatedAt != nil {
		result.CreatedAt = new(grpcutil.TimestampToTime(s.CreatedAt))
	}
	if s.UpdatedAt != nil {
		result.UpdatedAt = new(grpcutil.TimestampToTime(s.UpdatedAt))
	}
	return result
}
