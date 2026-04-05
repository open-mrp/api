package unitep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UnitSvc interface {
	ListUnits(ctx context.Context, req *ListUnitsRequest) (*apiresource.List[apiresource.Unit], *apierror.APIError)
	GetUnit(ctx context.Context, req *GetUnitRequest) (*apiresource.Unit, *apierror.APIError)
	CreateUnit(ctx context.Context, req *CreateUnitRequest) (*apiresource.Unit, *apierror.APIError)
	UpdateUnit(ctx context.Context, req *UpdateUnitRequest) (*apiresource.Unit, *apierror.APIError)
	DeleteUnit(ctx context.Context, req *DeleteUnitRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ValidateUnits(ctx context.Context, req *ValidateUnitsRequest) (*apiresource.ValidateUnitsResponse, *apierror.APIError)
}

type UnitSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type unitSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var unitSvcTracer = tracing.GetTracer("api-gateway.endpoints.units.service")

func (c *UnitSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("unit endpoint service: core client is required")
	}
	return nil
}

func NewUnitSvc(config *UnitSvcConfig) UnitSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &unitSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *unitSvcImpl) ListUnits(ctx context.Context, req *ListUnitsRequest) (*apiresource.List[apiresource.Unit], *apierror.APIError) {
	var unitType *string
	if req.Type != nil {
		s := string(*req.Type)
		unitType = &s
	}

	pbReq := &pb.ListUnitsRequest{
		Cursor:       req.Cursor,
		Limit:        req.Limit,
		Query:        req.Query,
		Type:         unitType,
		UnitGroupIds: req.UnitGroupIDs,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitSvcTracer, "service.units.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListUnitsResponse, error) {
			return m.coreClient.ListUnits(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return UnitListPresenter(resp), nil
}

func (m *unitSvcImpl) GetUnit(ctx context.Context, req *GetUnitRequest) (*apiresource.Unit, *apierror.APIError) {
	pbReq := &pb.GetUnitRequest{
		Id: req.UnitID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitSvcTracer, "service.units.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetUnitResponse, error) {
			return m.coreClient.GetUnit(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := UnitPresenter(resp.Unit)
	return &result, nil
}

func (m *unitSvcImpl) CreateUnit(ctx context.Context, req *CreateUnitRequest) (*apiresource.Unit, *apierror.APIError) {
	pbReq := &pb.CreateUnitRequest{
		Name:              req.Name,
		Abbreviation:      req.Abbreviation,
		Type:              string(req.Type),
		RatioNumerator:    req.RatioNumerator,
		RatioDenominator:  req.RatioDenominator,
		OffsetNumerator:   req.OffsetNumerator,
		OffsetDenominator: req.OffsetDenominator,
		IsBaseUnit:        false,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitSvcTracer, "service.units.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateUnitResponse, error) {
			return m.coreClient.CreateUnit(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := UnitPresenter(resp.Unit)
	return &result, nil
}

func (m *unitSvcImpl) UpdateUnit(ctx context.Context, req *UpdateUnitRequest) (*apiresource.Unit, *apierror.APIError) {
	pbReq := &pb.UpdateUnitRequest{
		Id:                req.UnitID,
		Name:              req.Name,
		Abbreviation:      req.Abbreviation,
		RatioNumerator:    req.RatioNumerator,
		RatioDenominator:  req.RatioDenominator,
		OffsetNumerator:   req.OffsetNumerator,
		OffsetDenominator: req.OffsetDenominator,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitSvcTracer, "service.units.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateUnitResponse, error) {
			return m.coreClient.UpdateUnit(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := UnitPresenter(resp.Unit)
	return &result, nil
}

func (m *unitSvcImpl) DeleteUnit(ctx context.Context, req *DeleteUnitRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteUnitRequest{
		Id: req.UnitID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, unitSvcTracer, "service.units.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteUnit(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *unitSvcImpl) ValidateUnits(ctx context.Context, req *ValidateUnitsRequest) (*apiresource.ValidateUnitsResponse, *apierror.APIError) {
	pbReq := &pb.ValidateUnitsRequest{
		UnitMap: req.UnitMap,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, unitSvcTracer, "service.units.validate", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ValidateUnitsResponse, error) {
			return m.coreClient.ValidateUnits(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ValidateUnitsPresenter(resp), nil
}
