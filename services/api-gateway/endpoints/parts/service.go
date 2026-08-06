package partep

import (
	"context"
	"fmt"

	jobep "github.com/augno/api/services/api-gateway/endpoints/jobs"
	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
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
	BulkUpsertParts(ctx context.Context, req *BulkUpsertPartsRequest) (*apiresource.Job, *apierror.APIError)
	UpdatePart(ctx context.Context, req *UpdatePartRequest) (*apiresource.Part, *apierror.APIError)
	DeletePart(ctx context.Context, req *DeletePartRequest) (*apiresource.Part, *apierror.APIError)
	ExportParts(ctx context.Context, req *ExportPartsRequest) (*apiresource.Job, *apierror.APIError)
}

type PartSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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

func (m *partSvcImpl) BulkUpsertParts(ctx context.Context, req *BulkUpsertPartsRequest) (*apiresource.Job, *apierror.APIError) {
	pbParts := make([]*pb.UpsertPartInput, len(req.Parts))
	for i, p := range req.Parts {
		pbProps := make([]*pb.UpsertItemPropertyInput, len(p.Properties))
		for j, pr := range p.Properties {
			pbProps[j] = &pb.UpsertItemPropertyInput{Name: pr.Name, Value: pr.Value}
		}
		pbParts[i] = &pb.UpsertPartInput{
			Sku:         p.SKU,
			Description: p.Description.Ptr(),
			Notes:       p.Notes.Ptr(),
			Category:    apirequest.ObjectIdentifierToProto(p.Category),
			UnitPrice:   rateInputToProto(p.UnitPrice.Ptr()),
			UnitCost:    rateInputToProto(p.UnitCost.Ptr()),
			Properties:  pbProps,
		}
	}

	pbReq := &pb.BulkUpsertPartsRequest{Parts: pbParts}

	resp, apiErr := grpcutil.CallRPC(ctx, partSvcTracer, "service.parts.bulk_upsert", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BulkUpsertPartsResponse, error) {
			return m.coreClient.BulkUpsertParts(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return jobep.JobFromProto(resp.GetJob()), nil
}

func (m *partSvcImpl) UpdatePart(ctx context.Context, req *UpdatePartRequest) (*apiresource.Part, *apierror.APIError) {
	pbReq := &pb.UpdatePartRequest{
		Id:          req.ItemID,
		Sku:         req.SKU.Ptr(),
		Description: field.StringClearableToProto(req.Description),
		Notes:       field.StringClearableToProto(req.Notes),
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

	// Gated build: the expandable item stays nil and populates only when the caller requests it via ?include=item. PartPresenter would embed it unconditionally and is reserved for the Excel export path.
	result := resourceloaders.PartFromProto(resp.Part)
	resourceloaders.StashPartMeta(ctx, resp.Part)
	return result, nil
}

func (m *partSvcImpl) ExportParts(ctx context.Context, req *ExportPartsRequest) (*apiresource.Job, *apierror.APIError) {
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

	return jobep.JobFromProto(resp.GetJob()), nil
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
