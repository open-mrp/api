package partep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	"github.com/augno/api/services/api-gateway/internal/export"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	httptransport "github.com/augno/api/services/api-gateway/internal/http"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
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
		Includes:     appctx.GetRequestedIncludeKeys(ctx),
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

	return PartListPresenter(resp), nil
}

func (m *partSvcImpl) GetPart(ctx context.Context, req *RetrievePartRequest) (*apiresource.Part, *apierror.APIError) {
	pbReq := &pb.GetPartRequest{
		Id:       req.ItemID,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, partSvcTracer, "service.parts.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPartResponse, error) {
			return m.coreClient.GetPart(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := PartPresenter(resp.Part)
	return &result, nil
}

func (m *partSvcImpl) CreatePart(ctx context.Context, req *CreatePartRequest) (*apiresource.Part, *apierror.APIError) {
	pbReq := &pb.CreatePartRequest{
		Sku:          req.SKU,
		Description:  req.Description,
		Notes:        req.Notes,
		CategoryId:   req.CategoryID,
		UnitPrice:    rateInputToProto(req.UnitPrice),
		UnitCost:     rateInputToProto(req.UnitCost),
		BurnRate:     rateInputToProto(req.BurnRate),
		AttributeIds: req.AttributeIDs,
		Includes:     appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, partSvcTracer, "service.parts.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreatePartResponse, error) {
			return m.coreClient.CreatePart(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := PartPresenter(resp.Part)
	return &result, nil
}

func (m *partSvcImpl) UpdatePart(ctx context.Context, req *UpdatePartRequest) (*apiresource.Part, *apierror.APIError) {
	pbReq := &pb.UpdatePartRequest{
		Id:       req.ItemID,
		Sku:      req.SKU,
		Includes: appctx.GetRequestedIncludeKeys(ctx),
	}

	if req.Description != nil {
		pbReq.UpdateDescription = true
		pbReq.Description = req.Description
	}

	if req.Notes != nil {
		pbReq.UpdateNotes = true
		pbReq.Notes = req.Notes
	}

	resp, apiErr := grpcutil.CallRPC(ctx, partSvcTracer, "service.parts.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdatePartResponse, error) {
			return m.coreClient.UpdatePart(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := PartPresenter(resp.Part)
	return &result, nil
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
	const pageSize = int32(500)

	pbReq := &pb.ListPartsRequest{
		Limit:        pageSize,
		Query:        req.Query,
		CategoryIds:  req.CategoryIDs,
		AttributeIds: req.AttributeIDs,
		Includes: []string{
			"item",
			"item.category",
			"item.category.properties",
			"item.unit_value",
			"item.unit_cost",
			"item.attributes",
		},
	}
	if req.StartDate != nil {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	var allParts []apiresource.Part
	for {
		resp, apiErr := grpcutil.CallRPC(ctx, partSvcTracer, "service.parts.export.page", domain.ServiceName,
			func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPartsResponse, error) {
				return m.coreClient.ListParts(ctx, pbReq, opts...)
			})
		if apiErr != nil {
			return nil, apiErr
		}
		for _, p := range resp.Parts {
			allParts = append(allParts, PartPresenter(p))
		}
		if !resp.PageInfo.HasNextPage {
			break
		}
		pbReq.Cursor = resp.PageInfo.NextCursor
	}

	body, err := export.PartsToExcel(allParts)
	if err != nil {
		return nil, apierror.NewInternalError(err, "Failed to build export file.")
	}

	return &httptransport.FileDownload{
		ContentType: export.ExcelContentType,
		Filename:    "parts.xlsx",
		Body:        body,
	}, nil
}
