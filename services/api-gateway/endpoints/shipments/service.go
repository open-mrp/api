package shipmentep

import (
	"context"
	"fmt"
	"strconv"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/appctx"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/core"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ShipmentSvc interface {
	ListShipments(ctx context.Context, req *ListShipmentsRequest) (*apiresource.List[apiresource.ShipmentSummary], *apierror.APIError)
	GetShipment(ctx context.Context, req *RetrieveShipmentRequest) (*apiresource.ShipmentDetail, *apierror.APIError)
	UpdateShipment(ctx context.Context, req *UpdateShipmentRequest) (*apiresource.ShipmentDetail, *apierror.APIError)
	DeleteShipment(ctx context.Context, req *DeleteShipmentRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ShipShipment(ctx context.Context, req *ShipShipmentRequest) (*apiresource.ShipmentDetail, *apierror.APIError)
	VoidShipment(ctx context.Context, req *VoidShipmentRequest) (*apiresource.ShipmentDetail, *apierror.APIError)
	EstimateRate(ctx context.Context, req *EstimateRateRequest) (*apiresource.EstimateRateResult, *apierror.APIError)
	RateShop(ctx context.Context, req *RateShopRequest) (*apiresource.RateShopResult, *apierror.APIError)
	ListShipmentLines(ctx context.Context, req *ListShipmentLinesRequest) (*apiresource.List[apiresource.ShipmentLine], *apierror.APIError)
	GetShipmentLine(ctx context.Context, req *RetrieveShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError)
	CreateShipmentLine(ctx context.Context, req *CreateShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError)
	UpdateShipmentLine(ctx context.Context, req *UpdateShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError)
	DeleteShipmentLine(ctx context.Context, req *DeleteShipmentLineRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ShipmentSvcConfig struct {
	CoreClient pb.CoreShippingServiceClient
}

type shipmentSvcImpl struct {
	coreClient pb.CoreShippingServiceClient
}

var shipmentSvcTracer = tracing.GetTracer("api-gateway.endpoints.shipments.service")

func (c *ShipmentSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("shipment endpoint service: core client is required")
	}
	return nil
}

func NewShipmentSvc(config *ShipmentSvcConfig) ShipmentSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &shipmentSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *shipmentSvcImpl) ListShipments(ctx context.Context, req *ListShipmentsRequest) (*apiresource.List[apiresource.ShipmentSummary], *apierror.APIError) {
	pbReq := &pb.ListShipmentsRequest{
		Cursor:           req.Cursor,
		Limit:            req.Limit,
		Query:            req.Query,
		Status:           req.Status,
		ItemIds:          req.ItemIDs,
		CustomerIds:      req.CustomerIDs,
		ProductLineIds:   req.ProductLineIDs,
		CustomerGroupIds: req.CustomerGroupIDs,
		SalesRepIds:      req.SalesRepIDs,
		StartDate:        req.StartDate,
		EndDate:          req.EndDate,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListShipmentsResponse, error) {
			return m.coreClient.ListShipments(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ShipmentListPresenter(ctx, resp), nil
}

func (m *shipmentSvcImpl) GetShipment(ctx context.Context, req *RetrieveShipmentRequest) (*apiresource.ShipmentDetail, *apierror.APIError) {
	pbReq := &pb.GetShipmentRequest{
		Id:       req.ShipmentID,
		Includes: req.Includes,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetShipmentResponse, error) {
			return m.coreClient.GetShipment(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ShipmentPresenter(resp.Shipment)
	return &result, nil
}

func (m *shipmentSvcImpl) UpdateShipment(ctx context.Context, req *UpdateShipmentRequest) (*apiresource.ShipmentDetail, *apierror.APIError) {
	pbReq := &pb.UpdateShipmentRequest{
		Id:                   req.ShipmentID,
		Note:                 req.Note,
		Number:               req.Number,
		MasterTrackingNumber: req.MasterTrackingNumber,
		CarrierId:            req.CarrierID,
		ServiceLevelId:       req.ServiceLevelID,
		Includes:             appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateShipmentResponse, error) {
			return m.coreClient.UpdateShipment(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ShipmentPresenter(resp.Shipment)
	return &result, nil
}

func (m *shipmentSvcImpl) DeleteShipment(ctx context.Context, req *DeleteShipmentRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteShipmentRequest{
		Id: req.ShipmentID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteShipment(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *shipmentSvcImpl) ShipShipment(ctx context.Context, req *ShipShipmentRequest) (*apiresource.ShipmentDetail, *apierror.APIError) {
	pbReq := &pb.ShipShipmentRequest{
		Id:            req.ShipmentID,
		EmailCustomer: req.EmailCustomer,
		Includes:      appctx.GetRequestedIncludeKeys(ctx),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.ship", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ShipShipmentResponse, error) {
			return m.coreClient.ShipShipment(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ShipmentPresenter(resp.Shipment)
	return &result, nil
}

func (m *shipmentSvcImpl) VoidShipment(ctx context.Context, req *VoidShipmentRequest) (*apiresource.ShipmentDetail, *apierror.APIError) {
	pbReq := &pb.VoidShipmentRequest{
		Id: req.ShipmentID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.void", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.VoidShipmentResponse, error) {
			return m.coreClient.VoidShipment(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ShipmentPresenter(resp.Shipment)
	return &result, nil
}

func (m *shipmentSvcImpl) EstimateRate(ctx context.Context, req *EstimateRateRequest) (*apiresource.EstimateRateResult, *apierror.APIError) {
	parcels := make([]*pb.ParcelInfo, len(req.Parcels))
	for i, p := range req.Parcels {
		parcels[i] = &pb.ParcelInfo{
			Weight: strconv.FormatFloat(p.Weight, 'f', -1, 64),
			Length: strconv.FormatFloat(p.Length, 'f', -1, 64),
			Width:  strconv.FormatFloat(p.Width, 'f', -1, 64),
			Height: strconv.FormatFloat(p.Height, 'f', -1, 64),
		}
	}

	pbReq := &pb.EstimateRateRequest{
		CarrierId:      req.CarrierID,
		ServiceLevelId: req.ServiceLevelID,
		ProductLineIds: req.ProductLineIDs,
		CustomerId:     req.CustomerID,
		From:           addressInputToProto(req.FromAddress),
		To:             addressInputToProto(req.ToAddress),
		Parcels:        parcels,
		OrderTotal:     req.OrderTotal,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.estimate_rate", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.EstimateRateResponse, error) {
			return m.coreClient.EstimateRate(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return EstimateRatePresenter(resp), nil
}

func (m *shipmentSvcImpl) RateShop(ctx context.Context, req *RateShopRequest) (*apiresource.RateShopResult, *apierror.APIError) {
	parcels := make([]*pb.ParcelInfo, len(req.Parcels))
	for i, p := range req.Parcels {
		parcels[i] = &pb.ParcelInfo{
			Weight: strconv.FormatFloat(p.Weight, 'f', -1, 64),
			Length: strconv.FormatFloat(p.Length, 'f', -1, 64),
			Width:  strconv.FormatFloat(p.Width, 'f', -1, 64),
			Height: strconv.FormatFloat(p.Height, 'f', -1, 64),
		}
	}

	pbReq := &pb.RateShopRequest{
		ProductLineIds: req.ProductLineIDs,
		CustomerId:     req.CustomerID,
		From:           addressInputToProto(req.FromAddress),
		To:             addressInputToProto(req.ToAddress),
		Parcels:        parcels,
		OrderTotal:     req.OrderTotal,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.rate_shop", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.RateShopResponse, error) {
			return m.coreClient.RateShop(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return RateShopPresenter(resp), nil
}

func (m *shipmentSvcImpl) ListShipmentLines(ctx context.Context, req *ListShipmentLinesRequest) (*apiresource.List[apiresource.ShipmentLine], *apierror.APIError) {
	pbReq := &pb.ListShipmentLinesRequest{
		ShipmentId: req.ShipmentID,
		Cursor:     req.Cursor,
		Limit:      req.Limit,
	}
	if req.Query != nil {
		pbReq.Query = req.Query
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.list_lines", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListShipmentLinesResponse, error) {
			return m.coreClient.ListShipmentLines(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return ShipmentLineListPresenter(ctx, resp), nil
}

func (m *shipmentSvcImpl) GetShipmentLine(ctx context.Context, req *RetrieveShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError) {
	pbReq := &pb.GetShipmentLineRequest{
		ShipmentId: req.ShipmentID,
		Id:         req.ShipmentLineID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.get_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetShipmentLineResponse, error) {
			return m.coreClient.GetShipmentLine(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ShipmentLinePresenter(resp.ShipmentLine)
	return &result, nil
}

func (m *shipmentSvcImpl) CreateShipmentLine(ctx context.Context, req *CreateShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError) {
	pbReq := &pb.CreateShipmentLineRequest{
		ShipmentId:       req.ShipmentID,
		SalesOrderLineId: req.SalesOrderLineID,
		QuantityValue:    req.QuantityValue,
		QuantityUnitId:   req.QuantityUnitID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.create_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateShipmentLineResponse, error) {
			return m.coreClient.CreateShipmentLine(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ShipmentLinePresenter(resp.ShipmentLine)
	return &result, nil
}

func (m *shipmentSvcImpl) UpdateShipmentLine(ctx context.Context, req *UpdateShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError) {
	pbReq := &pb.UpdateShipmentLineRequest{
		ShipmentId:     req.ShipmentID,
		Id:             req.ShipmentLineID,
		QuantityValue:  req.QuantityValue,
		QuantityUnitId: req.QuantityUnitID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.update_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateShipmentLineResponse, error) {
			return m.coreClient.UpdateShipmentLine(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := ShipmentLinePresenter(resp.ShipmentLine)
	return &result, nil
}

func (m *shipmentSvcImpl) DeleteShipmentLine(ctx context.Context, req *DeleteShipmentLineRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteShipmentLineRequest{
		ShipmentId: req.ShipmentID,
		Id:         req.ShipmentLineID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.delete_line", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteShipmentLine(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func addressInputToProto(a apirequest.AddressInput) *pb.AddressInput {
	return &pb.AddressInput{
		Name:    a.Name,
		Street1: derefStr(a.StreetLine1.Ptr()),
		Street2: a.StreetLine2.Ptr(),
		City:    derefStr(a.Locality.Ptr()),
		State:   derefStr(a.State.Ptr()),
		Zip:     derefStr(a.PostalCode.Ptr()),
		Country: a.Country,
		Phone:   a.Phone.Ptr(),
		Email:   a.Email.Ptr(),
	}
}
