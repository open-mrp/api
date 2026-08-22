package shippingcaseep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

type ShippingCaseSvc interface {
	GetShippingCase(ctx context.Context, req *RetrieveShippingCaseRequest) (*apiresource.ShippingCase, *apierror.APIError)
	UpdateShippingCase(ctx context.Context, req *UpdateShippingCaseRequest) (*apiresource.ShippingCase, *apierror.APIError)
	AdminUpdateShippingCaseTracking(ctx context.Context, req *AdminUpdateShippingCaseTrackingRequest) (*apiresource.ShippingCase, *apierror.APIError)
	DeleteShippingCase(ctx context.Context, req *DeleteShippingCaseRequest) (*apiresource.EmptyResource, *apierror.APIError)
	GetShippingCaseLabel(ctx context.Context, req *GetShippingCaseLabelRequest) (*apiresource.ShippingCaseLabelURL, *apierror.APIError)
}

type ShippingCaseSvcConfig struct {
	// CoreClient (required) is the core-service shipping-case gRPC client.
	CoreClient pb.CoreShippingCaseServiceClient
}

type shippingCaseSvcImpl struct {
	coreClient pb.CoreShippingCaseServiceClient
}

var shippingCaseSvcTracer = tracing.GetTracer("api-gateway.endpoints.shipping-cases.service")

func (c *ShippingCaseSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("shipping case endpoint service: core client is required")
	}
	return nil
}

func NewShippingCaseSvc(config *ShippingCaseSvcConfig) ShippingCaseSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &shippingCaseSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *shippingCaseSvcImpl) GetShippingCase(ctx context.Context, req *RetrieveShippingCaseRequest) (*apiresource.ShippingCase, *apierror.APIError) {
	pbReq := &pb.GetShippingCaseRequest{
		Id: req.ShippingCaseID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shippingCaseSvcTracer, "service.shipping_cases.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetShippingCaseResponse, error) {
			return m.coreClient.GetShippingCase(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := shippingCaseFromProto(resp.ShippingCase)
	stashShippingCaseMeta(meta, resp.ShippingCase)
	return &result, nil
}

func (m *shippingCaseSvcImpl) UpdateShippingCase(ctx context.Context, req *UpdateShippingCaseRequest) (*apiresource.ShippingCase, *apierror.APIError) {
	pbReq := &pb.UpdateShippingCaseRequest{
		Id:                  req.ShippingCaseID,
		TrackingNumber:      req.TrackingNumber.Ptr(),
		FreightAmountValue:  req.FreightAmountValue.Ptr(),
		FreightAmountUnitId: req.FreightAmountUnitID.Ptr(),
		FreightWeightValue:  req.FreightWeightValue.Ptr(),
		FreightWeightUnitId: req.FreightWeightUnitID.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shippingCaseSvcTracer, "service.shipping_cases.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateShippingCaseResponse, error) {
			return m.coreClient.UpdateShippingCase(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := shippingCaseFromProto(resp.ShippingCase)
	stashShippingCaseMeta(meta, resp.ShippingCase)
	return &result, nil
}

func (m *shippingCaseSvcImpl) AdminUpdateShippingCaseTracking(ctx context.Context, req *AdminUpdateShippingCaseTrackingRequest) (*apiresource.ShippingCase, *apierror.APIError) {
	pbReq := &pb.AdminUpdateShippingCaseTrackingRequest{
		Id:             req.ShippingCaseID,
		TrackingNumber: req.TrackingNumber.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shippingCaseSvcTracer, "service.shipping_cases.admin_update_tracking", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AdminUpdateShippingCaseTrackingResponse, error) {
			return m.coreClient.AdminUpdateShippingCaseTracking(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := shippingCaseFromProto(resp.ShippingCase)
	stashShippingCaseMeta(meta, resp.ShippingCase)
	return &result, nil
}

func (m *shippingCaseSvcImpl) DeleteShippingCase(ctx context.Context, req *DeleteShippingCaseRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteShippingCaseRequest{
		Id: req.ShippingCaseID,
	}

	_, apiErr := grpcutil.CallRPC(ctx, shippingCaseSvcTracer, "service.shipping_cases.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.DeleteShippingCaseResponse, error) {
			return m.coreClient.DeleteShippingCase(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

func (m *shippingCaseSvcImpl) GetShippingCaseLabel(ctx context.Context, req *GetShippingCaseLabelRequest) (*apiresource.ShippingCaseLabelURL, *apierror.APIError) {
	pbReq := &pb.GetShippingCaseLabelRequest{
		Id: req.ShippingCaseID,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shippingCaseSvcTracer, "service.shipping_cases.get_label", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetShippingCaseLabelResponse, error) {
			return m.coreClient.GetShippingCaseLabel(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.ShippingCaseLabelURL{
		Object: constants.ObjectTypeShippingCaseLabelURL,
		URL:    resp.Url,
	}, nil
}

func shippingCaseFromProto(sc *pb.ShippingCaseInfo) apiresource.ShippingCase {
	if sc == nil {
		return apiresource.ShippingCase{}
	}

	result := apiresource.ShippingCase{
		ID:             sc.Id,
		Object:         constants.ObjectTypeShippingCase,
		Number:         sc.Number,
		SSCC:           sc.Sscc,
		TrackingNumber: sc.TrackingNumber,
		CreatedAt:      grpcutil.TimestampToTime(sc.CreatedAt),
		UpdatedAt:      grpcutil.TimestampToTime(sc.UpdatedAt),
	}

	if sc.ShippedAt != nil {
		t := grpcutil.TimestampToTime(sc.ShippedAt)
		result.ShippedAt = &t
	}

	return result
}

func stashShippingCaseMeta(meta *resourcekit.LoadMeta, sc *pb.ShippingCaseInfo) {
	if sc == nil {
		return
	}

	carrier := &apiresource.Carrier{
		ID:     sc.CarrierId,
		Object: constants.ObjectTypeCarrier,
		Name:   sc.CarrierName,
	}
	if sc.CarrierIsPortalEnabled != nil && *sc.CarrierIsPortalEnabled {
		carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityVisible
	} else {
		carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityHidden
	}
	if sc.CarrierCreatedAt != nil {
		carrier.CreatedAt = sc.CarrierCreatedAt.AsTime()
	}
	if sc.CarrierUpdatedAt != nil {
		carrier.UpdatedAt = sc.CarrierUpdatedAt.AsTime()
	}
	meta.Set(constants.ObjectTypeShippingCase, sc.Id, "carrier", carrier)

	meta.Set(constants.ObjectTypeShippingCase, sc.Id, "shipment", &apiresource.Shipment{
		ID:        sc.ShipmentId,
		Object:    constants.ObjectTypeShipment,
		Number:    sc.GetShipmentNumber(),
		Status:    constants.ShipmentStatus(sc.GetShipmentStatusCode()),
		CreatedAt: grpcutil.TimestampToTime(sc.ShipmentCreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(sc.ShipmentUpdatedAt),
	})

	meta.Set(constants.ObjectTypeShippingCase, sc.Id, "freight_amount", &apiresource.Quantity{
		ID:           sc.FreightAmountId,
		Object:       constants.ObjectTypeQuantity,
		Value:        sc.FreightAmountValue,
		DisplayValue: apiresource.FormatDisplayValue(sc.FreightAmountValue, sc.FreightAmountUnitAbbreviation, sc.FreightAmountUnitType),
		Unit: &apiresource.Unit{
			ID:                sc.FreightAmountUnitId,
			Object:            constants.ObjectTypeUnit,
			Name:              sc.FreightAmountUnitName,
			Abbreviation:      sc.FreightAmountUnitAbbreviation,
			Type:              constants.UnitType(sc.FreightAmountUnitType),
			RatioNumerator:    sc.FreightAmountUnitRatioNumerator,
			RatioDenominator:  sc.FreightAmountUnitRatioDenominator,
			OffsetNumerator:   sc.FreightAmountUnitOffsetNumerator,
			OffsetDenominator: sc.FreightAmountUnitOffsetDenominator,
			CreatedAt:         grpcutil.TimestampToTime(sc.FreightAmountUnitCreatedAt),
			UpdatedAt:         grpcutil.TimestampToTime(sc.FreightAmountUnitUpdatedAt),
		},
	})

	meta.Set(constants.ObjectTypeShippingCase, sc.Id, "freight_weight", &apiresource.Quantity{
		ID:           sc.FreightWeightId,
		Object:       constants.ObjectTypeQuantity,
		Value:        sc.FreightWeightValue,
		DisplayValue: apiresource.FormatDisplayValue(sc.FreightWeightValue, sc.FreightWeightUnitAbbreviation, sc.FreightWeightUnitType),
		Unit: &apiresource.Unit{
			ID:                sc.FreightWeightUnitId,
			Object:            constants.ObjectTypeUnit,
			Name:              sc.FreightWeightUnitName,
			Abbreviation:      sc.FreightWeightUnitAbbreviation,
			Type:              constants.UnitType(sc.FreightWeightUnitType),
			RatioNumerator:    sc.FreightWeightUnitRatioNumerator,
			RatioDenominator:  sc.FreightWeightUnitRatioDenominator,
			OffsetNumerator:   sc.FreightWeightUnitOffsetNumerator,
			OffsetDenominator: sc.FreightWeightUnitOffsetDenominator,
			CreatedAt:         grpcutil.TimestampToTime(sc.FreightWeightUnitCreatedAt),
			UpdatedAt:         grpcutil.TimestampToTime(sc.FreightWeightUnitUpdatedAt),
		},
	})
}
