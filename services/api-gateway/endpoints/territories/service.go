package territoryep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type TerritorySvc interface {
	ListTerritories(ctx context.Context, req *ListTerritoriesRequest) (*apiresource.List[apiresource.Territory], *apierror.APIError)
	GetTerritory(ctx context.Context, req *RetrieveTerritoryRequest) (*apiresource.Territory, *apierror.APIError)
	CreateTerritory(ctx context.Context, req *CreateTerritoryRequest) (*apiresource.Territory, *apierror.APIError)
	UpdateTerritory(ctx context.Context, req *UpdateTerritoryRequest) (*apiresource.Territory, *apierror.APIError)
	DeleteTerritory(ctx context.Context, req *DeleteTerritoryRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type TerritorySvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type territorySvcImpl struct {
	coreClient pb.CoreServiceClient
}

var territorySvcTracer = tracing.GetTracer("api-gateway.endpoints.territories.service")

func (c *TerritorySvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("territory endpoint service: core client is required")
	}
	return nil
}

func NewTerritorySvc(config *TerritorySvcConfig) TerritorySvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &territorySvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *territorySvcImpl) ListTerritories(ctx context.Context, req *ListTerritoriesRequest) (*apiresource.List[apiresource.Territory], *apierror.APIError) {
	pbReq := &pb.ListTerritoriesRequest{
		AccountId: req.AccountID,
		Cursor:    req.Cursor,
		Limit:     req.Limit,
		Query:     req.Query,
		Includes:  appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, territorySvcTracer, "service.territories.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListTerritoriesResponse, error) {
			return m.coreClient.ListTerritories(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return TerritoryListPresenter(ctx, resp), nil
}

func (m *territorySvcImpl) GetTerritory(ctx context.Context, req *RetrieveTerritoryRequest) (*apiresource.Territory, *apierror.APIError) {
	pbReq := &pb.GetTerritoryRequest{
		AccountId: req.AccountID,
		Id:        req.TerritoryID,
		Includes:  appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, territorySvcTracer, "service.territories.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetTerritoryResponse, error) {
			return m.coreClient.GetTerritory(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := TerritoryPresenter(resp.Territory)
	return &result, nil
}

func (m *territorySvcImpl) CreateTerritory(ctx context.Context, req *CreateTerritoryRequest) (*apiresource.Territory, *apierror.APIError) {
	pbReq := &pb.CreateTerritoryRequest{
		AccountId:     req.AccountID,
		State:         req.State,
		StartZipcode:  req.StartZipcode,
		EndZipcode:    req.EndZipcode,
		SalesRepId:    req.SalesRepID,
		ProductLineId: req.ProductLineID,
		Includes:      appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, territorySvcTracer, "service.territories.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateTerritoryResponse, error) {
			return m.coreClient.CreateTerritory(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := TerritoryPresenter(resp.Territory)
	return &result, nil
}

func (m *territorySvcImpl) UpdateTerritory(ctx context.Context, req *UpdateTerritoryRequest) (*apiresource.Territory, *apierror.APIError) {
	pbReq := &pb.UpdateTerritoryRequest{
		AccountId:         req.AccountID,
		Id:                req.TerritoryID,
		State:             req.State,
		StartZipcode:      req.StartZipcode,
		EndZipcode:        req.EndZipcode,
		SalesRepId:        req.SalesRepID,
		ProductLineId:     req.ProductLineID,
		ClearProductLine:  derefBool(req.ClearProductLine),
		ClearStartZipcode: derefBool(req.ClearStartZipcode),
		ClearEndZipcode:   derefBool(req.ClearEndZipcode),
		Includes:          appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, territorySvcTracer, "service.territories.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateTerritoryResponse, error) {
			return m.coreClient.UpdateTerritory(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := TerritoryPresenter(resp.Territory)
	return &result, nil
}

func (m *territorySvcImpl) DeleteTerritory(ctx context.Context, req *DeleteTerritoryRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteTerritoryRequest{
		AccountId: req.AccountID,
		Id:        req.TerritoryID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, territorySvcTracer, "service.territories.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteTerritory(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
