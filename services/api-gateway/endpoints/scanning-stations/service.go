package scanningstationep

import (
	"context"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ScanningStationSvc interface {
	ListScanningStations(ctx context.Context, req *ListScanningStationsRequest) (*apiresource.List[apiresource.ScanningStation], *apierror.APIError)
	GetScanningStation(ctx context.Context, req *RetrieveScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError)
	CreateScanningStation(ctx context.Context, req *CreateScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError)
	UpdateScanningStation(ctx context.Context, req *UpdateScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError)
	DeleteScanningStation(ctx context.Context, req *DeleteScanningStationRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ConnectProductionSteps(ctx context.Context, req *ConnectProductionStepsRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ScanningStationSvcConfig struct {
	// CoreClient (required) is the core-service gRPC client.
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

func loadScanningStationByID(ctx context.Context, id string) (*apiresource.ScanningStation, *apierror.APIError) {
	loaded, apiErr := resourceloaders.LoadScanningStations(ctx, []string{id})
	if apiErr != nil {
		return nil, apiErr
	}
	v, ok := loaded[id]
	if !ok {
		return nil, apierror.NewResourceNotFoundError("Scanning station not found.")
	}
	return v.(*apiresource.ScanningStation), nil
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

	ids := make([]string, len(resp.ScanningStations))
	for i, ss := range resp.ScanningStations {
		ids[i] = ss.Id
	}

	loaded, apiErr := resourceloaders.LoadScanningStations(ctx, ids)
	if apiErr != nil {
		return nil, apiErr
	}

	stations := make([]apiresource.ScanningStation, 0, len(resp.ScanningStations))
	for _, ss := range resp.ScanningStations {
		if v, ok := loaded[ss.Id]; ok {
			stations = append(stations, *v.(*apiresource.ScanningStation))
		}
	}

	return apiresource.NewList(stations, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *scanningStationSvcImpl) GetScanningStation(ctx context.Context, req *RetrieveScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError) {
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

	return loadScanningStationByID(ctx, resp.ScanningStation.Id)
}

func (m *scanningStationSvcImpl) CreateScanningStation(ctx context.Context, req *CreateScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError) {
	pbReq := &pb.CreateScanningStationRequest{
		Name:                req.Name,
		Notes:               req.Notes.Ptr(),
		Type:                string(req.Type),
		OperatorRequirement: string(req.OperatorRequirement),
		DepartmentId:        req.DepartmentID,
		LabelSizeCode:       req.LabelSizeCode.Ptr().StringPtr(),
		LabelTypeCode:       req.LabelTypeCode.Ptr().StringPtr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, scanningStationSvcTracer, "service.scanning_stations.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateScanningStationResponse, error) {
			return m.coreClient.CreateScanningStation(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return loadScanningStationByID(ctx, resp.ScanningStation.Id)
}

func (m *scanningStationSvcImpl) UpdateScanningStation(ctx context.Context, req *UpdateScanningStationRequest) (*apiresource.ScanningStation, *apierror.APIError) {
	pbReq := &pb.UpdateScanningStationRequest{
		Id:            req.ScanningStationID,
		Name:          req.Name.Ptr(),
		Notes:         field.StringClearableToProto(req.Notes),
		LabelSizeCode: req.LabelSizeCode.Ptr().StringPtr(),
		LabelTypeCode: req.LabelTypeCode.Ptr().StringPtr(),
		OperatorRequirement: func() *string {
			if v, ok := req.OperatorRequirement.Value(); ok {
				s := string(v)
				return &s
			}
			return nil
		}(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, scanningStationSvcTracer, "service.scanning_stations.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateScanningStationResponse, error) {
			return m.coreClient.UpdateScanningStation(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return loadScanningStationByID(ctx, resp.ScanningStation.Id)
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
