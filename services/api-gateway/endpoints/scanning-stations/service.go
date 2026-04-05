package scanningstationep

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

type ScanningStationSvc interface {
	ListScanningStations(ctx context.Context, req *ListScanningStationsRequest) (*apiresource.List[apiresource.ScanningStation], *apierror.APIError)
	GetScanningStation(ctx context.Context, req *GetScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError)
	CreateScanningStation(ctx context.Context, req *CreateScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError)
	UpdateScanningStation(ctx context.Context, req *UpdateScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError)
	DeleteScanningStation(ctx context.Context, req *DeleteScanningStationRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ConnectProductionSteps(ctx context.Context, req *ConnectProductionStepsRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ScanningStationSvcConfig struct {
	CoreClient pb.CoreServiceClient
}

type scanningStationSvcImpl struct {
	coreClient pb.CoreServiceClient
}

var scanningStationSvcTracer = tracing.GetTracer("api-gateway.endpoints.scanning_stations.service")

func (c *ScanningStationSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("scanning station endpoint service: core client is required")
	}
	return nil
}

func NewScanningStationSvc(config *ScanningStationSvcConfig) ScanningStationSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &scanningStationSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *scanningStationSvcImpl) ListScanningStations(ctx context.Context, req *ListScanningStationsRequest) (*apiresource.List[apiresource.ScanningStation], *apierror.APIError) {
	pbReq := &pb.ListScanningStationsRequest{
		Cursor: req.Cursor,
		Limit:  req.Limit,
		Query:  req.Query,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, scanningStationSvcTracer, "service.scanning_stations.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListScanningStationsResponse, error) {
			return m.coreClient.ListScanningStations(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ScanningStationListPresenter(resp), nil
}

func (m *scanningStationSvcImpl) GetScanningStation(ctx context.Context, req *GetScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError) {
	pbReq := &pb.GetScanningStationRequest{
		Id: req.ScanningStationID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, scanningStationSvcTracer, "service.scanning_stations.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetScanningStationResponse, error) {
			return m.coreClient.GetScanningStation(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ScanningStationPresenter(resp.ScanningStation)
	return &result, nil
}

func (m *scanningStationSvcImpl) CreateScanningStation(ctx context.Context, req *CreateScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError) {
	pbReq := &pb.CreateScanningStationRequest{
		Name:                  req.Name,
		Notes:                 req.Notes,
		Type:                  string(req.Type),
		MaterialCheckRequired: req.MaterialCheckRequired,
		DepartmentId:          req.DepartmentID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, scanningStationSvcTracer, "service.scanning_stations.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateScanningStationResponse, error) {
			return m.coreClient.CreateScanningStation(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ScanningStationPresenter(resp.ScanningStation)
	return &result, nil
}

func (m *scanningStationSvcImpl) UpdateScanningStation(ctx context.Context, req *UpdateScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError) {
	pbReq := &pb.UpdateScanningStationRequest{
		Id:                    req.ScanningStationID,
		Name:                  req.Name,
		Notes:                 req.Notes,
		LabelSizeCode:         req.LabelSizeCode.StringPtr(),
		LabelTypeCode:         req.LabelTypeCode.StringPtr(),
		MaterialCheckRequired: req.MaterialCheckRequired,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, scanningStationSvcTracer, "service.scanning_stations.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateScanningStationResponse, error) {
			return m.coreClient.UpdateScanningStation(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ScanningStationPresenter(resp.ScanningStation)
	return &result, nil
}

func (m *scanningStationSvcImpl) DeleteScanningStation(ctx context.Context, req *DeleteScanningStationRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteScanningStationRequest{
		Id: req.ScanningStationID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, scanningStationSvcTracer, "service.scanning_stations.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteScanningStation(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *scanningStationSvcImpl) ConnectProductionSteps(ctx context.Context, req *ConnectProductionStepsRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.ConnectProductionStepsByScanningStationRequest{
		Id:   req.ScanningStationID,
		Name: req.Name,
	}

	_, apiErr := grpcutil.CallRPC(ctx, scanningStationSvcTracer, "service.scanning_stations.connect_production_steps", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.ConnectProductionStepsByScanningStation(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}
