package pickep

import (
	"context"
	"fmt"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"

	"github.com/augno/api/services/api-gateway/internal/domain"
)

var pickEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.picks.service")

type PickSvc interface {
	ListPicks(ctx context.Context, req *ListPicksRequest) (*apiresource.List[apiresource.PickSummary], *apierror.APIError)
	GetPick(ctx context.Context, req *RetrievePickRequest) (*apiresource.PickDetail, *apierror.APIError)
	UpdatePick(ctx context.Context, req *UpdatePickRequest) (*apiresource.PickDetail, *apierror.APIError)
	PickAllLines(ctx context.Context, req *PickAllLinesRequest) (*apiresource.PickDetail, *apierror.APIError)
	VoidPick(ctx context.Context, req *VoidPickRequest) (*apiresource.PickDetail, *apierror.APIError)
	PackPick(ctx context.Context, req *PackPickRequest) (*apiresource.PackPickResponse, *apierror.APIError)
	GetPickShipments(ctx context.Context, req *GetPickShipmentsRequest) (*apiresource.PickShipmentsResponse, *apierror.APIError)
	UpdatePickLine(ctx context.Context, req *UpdatePickLineRequest) (*apiresource.PickLineDetail, *apierror.APIError)
	PickPickLine(ctx context.Context, req *PickPickLineRequest) (*apiresource.PickLineDetail, *apierror.APIError)
	VoidPickLine(ctx context.Context, req *VoidPickLineRequest) (*apiresource.PickLineDetail, *apierror.APIError)
}

type PickSvcConfig struct {
	CoreClient pb.CorePickingServiceClient
}

func (c *PickSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("pick endpoint service: core client is required")
	}
	return nil
}

type pickSvcImpl struct {
	coreClient pb.CorePickingServiceClient
}

func NewPickSvc(config *PickSvcConfig) PickSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}
	return &pickSvcImpl{coreClient: config.CoreClient}
}

func (m *pickSvcImpl) ListPicks(ctx context.Context, req *ListPicksRequest) (*apiresource.List[apiresource.PickSummary], *apierror.APIError) {
	pbReq := &pb.ListPicksRequest{
		Limit: req.Limit,
	}
	if req.Cursor != nil {
		pbReq.Cursor = req.Cursor
	}
	if req.Query != nil {
		pbReq.Query = req.Query
	}
	if req.Status != nil {
		pbReq.Status = req.Status
	}
	if len(req.CustomerIDs) > 0 {
		pbReq.CustomerIds = req.CustomerIDs
	}
	if len(req.ProductLineIDs) > 0 {
		pbReq.ProductLineIds = req.ProductLineIDs
	}
	if len(req.CustomerGroupIDs) > 0 {
		pbReq.CustomerGroupIds = req.CustomerGroupIDs
	}
	if len(req.DepartmentIDs) > 0 {
		pbReq.DepartmentIds = req.DepartmentIDs
	}
	if req.StartDate != nil {
		pbReq.StartDate = req.StartDate
	}
	if req.EndDate != nil {
		pbReq.EndDate = req.EndDate
	}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListPicksResponse, error) {
			return m.coreClient.ListPicks(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return PickListPresenter(resp), nil
}

func (m *pickSvcImpl) GetPick(ctx context.Context, req *RetrievePickRequest) (*apiresource.PickDetail, *apierror.APIError) {
	pbReq := &pb.GetPickRequest{Id: req.PickID, Includes: req.Includes}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPickResponse, error) {
			return m.coreClient.GetPick(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := PickDetailPresenter(resp.Pick)
	return &result, nil
}

func (m *pickSvcImpl) UpdatePick(ctx context.Context, req *UpdatePickRequest) (*apiresource.PickDetail, *apierror.APIError) {
	pbReq := &pb.UpdatePickRequest{Id: req.PickID, Includes: appctx.GetRequestedIncludeKeys(ctx)}
	if req.Number != nil {
		pbReq.Number = req.Number
	}
	if req.FinishedAt != nil {
		pbReq.FinishedAt = req.FinishedAt
	}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdatePickResponse, error) {
			return m.coreClient.UpdatePick(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := PickDetailPresenter(resp.Pick)
	return &result, nil
}

func (m *pickSvcImpl) PickAllLines(ctx context.Context, req *PickAllLinesRequest) (*apiresource.PickDetail, *apierror.APIError) {
	pbReq := &pb.PickAllLinesRequest{Id: req.PickID}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.pick_all_lines", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PickAllLinesResponse, error) {
			return m.coreClient.PickAllLines(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := PickDetailPresenter(resp.Pick)
	return &result, nil
}

func (m *pickSvcImpl) VoidPick(ctx context.Context, req *VoidPickRequest) (*apiresource.PickDetail, *apierror.APIError) {
	pbReq := &pb.VoidPickRequest{Id: req.PickID}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.void", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.VoidPickResponse, error) {
			return m.coreClient.VoidPick(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := PickDetailPresenter(resp.Pick)
	return &result, nil
}

func (m *pickSvcImpl) PackPick(ctx context.Context, req *PackPickRequest) (*apiresource.PackPickResponse, *apierror.APIError) {
	pbReq := &pb.PackPickRequest{Id: req.PickID, ShipmentCaseCount: req.ShipmentCaseCount}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.pack", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PackPickResponse, error) {
			return m.coreClient.PackPick(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	pick := PickDetailPresenter(resp.Pick)
	return &apiresource.PackPickResponse{Pick: &pick, ShipmentNumber: resp.ShipmentNumber}, nil
}

func (m *pickSvcImpl) GetPickShipments(ctx context.Context, req *GetPickShipmentsRequest) (*apiresource.PickShipmentsResponse, *apierror.APIError) {
	pbReq := &pb.GetPickShipmentsRequest{Id: req.PickID}
	if req.Query != nil {
		pbReq.Query = req.Query
	}
	if req.Limit != nil {
		pbReq.Limit = req.Limit
	}
	if req.Offset != nil {
		pbReq.Offset = req.Offset
	}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.get_shipments", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetPickShipmentsResponse, error) {
			return m.coreClient.GetPickShipments(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.PickShipmentsResponse{
		ShipmentNumbers: resp.ShipmentNumbers,
		Count:           resp.Count,
	}, nil
}

func (m *pickSvcImpl) UpdatePickLine(ctx context.Context, req *UpdatePickLineRequest) (*apiresource.PickLineDetail, *apierror.APIError) {
	pbReq := &pb.UpdatePickLineRequest{PickId: req.PickID, Id: req.PickLineID}
	if req.QuantityValue != nil {
		pbReq.QuantityValue = req.QuantityValue
	}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.update_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdatePickLineResponse, error) {
			return m.coreClient.UpdatePickLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := PickLineDetailPresenter(resp.PickLine)
	return &result, nil
}

func (m *pickSvcImpl) PickPickLine(ctx context.Context, req *PickPickLineRequest) (*apiresource.PickLineDetail, *apierror.APIError) {
	pbReq := &pb.PickPickLineRequest{PickId: req.PickID, Id: req.PickLineID}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.pick_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.PickPickLineResponse, error) {
			return m.coreClient.PickPickLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := PickLineDetailPresenter(resp.PickLine)
	return &result, nil
}

func (m *pickSvcImpl) VoidPickLine(ctx context.Context, req *VoidPickLineRequest) (*apiresource.PickLineDetail, *apierror.APIError) {
	pbReq := &pb.VoidPickLineRequest{PickId: req.PickID, Id: req.PickLineID}

	resp, apiErr := grpcutil.CallRPC(ctx, pickEpSvcTracer, "service.picks.void_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.VoidPickLineResponse, error) {
			return m.coreClient.VoidPickLine(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	result := PickLineDetailPresenter(resp.PickLine)
	return &result, nil
}
