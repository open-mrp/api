package partep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/services/api-gateway/internal/export"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/patch"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func rateInputToProto(r *apirequest.RateInput) *pb.CreateRateInput {
	if r == nil {
		return nil
	}
	return &pb.CreateRateInput{
		Value:             r.Value,
		NumeratorUnitId:   r.NumeratorUnitID,
		DenominatorUnitId: r.DenominatorUnitID,
	}
}

type PartSvc interface {
	ListParts(ctx context.Context, req *ListPartsRequest) (*apiresource.List[apiresource.Part], *apierror.APIError)
	GetPart(ctx context.Context, req *RetrievePartRequest) (*apiresource.Part, *apierror.APIError)
	CreatePart(ctx context.Context, req *CreatePartRequest) (*apiresource.Part, *apierror.APIError)
	UpdatePart(ctx context.Context, req *UpdatePartRequest) (*apiresource.Part, *apierror.APIError)
	DeletePart(ctx context.Context, req *DeletePartRequest) (*apiresource.Part, *apierror.APIError)
	ExportParts(ctx context.Context, req *ExportPartsRequest) (*httptransport.FileDownload, *apierror.APIError)
}

type PartSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type partSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var partSvcTracer = tracing.GetTracer("api-gateway.endpoints.parts.service")

func (c *PartSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("part endpoint service: core client is required")
	}
	return nil
}

func NewPartSvc(config *PartSvcConfig) PartSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &partSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *partSvcImpl) ListParts(ctx context.Context, req *ListPartsRequest) (*apiresource.List[apiresource.Part], *apierror.APIError) {
	pbReq := &pb.ListPartsRequest{
		Cursor:       req.Cursor,
		Limit:        req.Limit,
		Query:        req.Query,
		CategoryIds:  req.CategoryIDs,
		AttributeIds: req.AttributeIDs,
	}

	if req.StartDate != nil {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, partSvcTracer, "service.parts.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPartsResponse, error) {
			return m.coreClient.ListParts(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	ids := make([]string, len(resp.Parts))
	for i, p := range resp.Parts {
		ids[i] = p.Id
	}
	loaded, apiErr := resourceloaders.LoadParts(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}
	parts := make([]apiresource.Part, 0, len(ids))
	for _, id := range ids {
		if v, ok := loaded[id]; ok {
			parts = append(parts, *(v.(*apiresource.Part)))
		}
	}
	return apiresource.NewList(parts, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *partSvcImpl) GetPart(ctx context.Context, req *RetrievePartRequest) (*apiresource.Part, *apierror.APIError) {
	return loadPartByID(ctx, req.ItemID)
}

func (m *partSvcImpl) CreatePart(ctx context.Context, req *CreatePartRequest) (*apiresource.Part, *apierror.APIError) {
	pbReq := &pb.CreatePartRequest{
		Sku:          req.SKU,
		Description:  req.Description.Ptr(),
		Notes:        req.Notes.Ptr(),
		CategoryId:   req.CategoryID,
		UnitPrice:    rateInputToProto(req.UnitPrice.Ptr()),
		UnitCost:     rateInputToProto(req.UnitCost.Ptr()),
		BurnRate:     rateInputToProto(req.BurnRate.Ptr()),
		AttributeIds: req.AttributeIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, partSvcTracer, "service.parts.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreatePartResponse, error) {
			return m.coreClient.CreatePart(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return loadPartByID(ctx, resp.Part.Id)
}

func (m *partSvcImpl) UpdatePart(ctx context.Context, req *UpdatePartRequest) (*apiresource.Part, *apierror.APIError) {
	pbReq := &pb.UpdatePartRequest{
		Id:          req.ItemID,
		Sku:         req.SKU,
		Description: patch.StringFieldPtrToProto(req.Description),
		Notes:       patch.StringFieldPtrToProto(req.Notes),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, partSvcTracer, "service.parts.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdatePartResponse, error) {
			return m.coreClient.UpdatePart(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return loadPartByID(ctx, resp.Part.Id)
}

func (m *partSvcImpl) DeletePart(ctx context.Context, req *DeletePartRequest) (*apiresource.Part, *apierror.APIError) {
	pbReq := &pb.DeletePartRequest{
		Id: req.ItemID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, partSvcTracer, "service.parts.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeletePartResponse, error) {
			return m.coreClient.DeletePart(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := PartPresenter(resp.Part)
	return &result, nil
}

func (m *partSvcImpl) ExportParts(ctx context.Context, req *ExportPartsRequest) (*httptransport.FileDownload, *apierror.APIError) {
	pbReq := &pb.ExportPartsRequest{
		Query:        req.Query,
		CategoryIds:  req.CategoryIDs,
		AttributeIds: req.AttributeIDs,
	}
	if req.StartDate != nil {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, partSvcTracer, "service.parts.export", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ExportPartsResponse, error) {
			return m.coreClient.ExportParts(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	parts := make([]apiresource.Part, len(resp.Parts))
	for i, p := range resp.Parts {
		parts[i] = PartPresenter(p)
	}

	body, err := export.PartsToExcel(parts)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to build export file.")
	}

	return &httptransport.FileDownload{
		ContentType: export.ExcelContentType,
		Filename:    "parts.xlsx",
		Body:        body,
	}, nil
}

func loadPartByID(ctx context.Context, id string) (*apiresource.Part, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadParts(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Part not found.")
	}
	return v.(*apiresource.Part), nil
}
