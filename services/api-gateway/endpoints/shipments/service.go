package shipmentep

import (
	"context"
	"fmt"
	"strconv"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
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

var shipmentIncludes = []string{"lines", "shipping_cases", "sales_order", "customer", "carrier", "service_level", "shipping_address", "shipped_by", "invoice", "pick"}

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

	if resp == nil {
		return apiresource.NewList[apiresource.ShipmentSummary](nil, apiresource.PageInfo{}), nil
	}

	summaries := make([]apiresource.ShipmentSummary, len(resp.Shipments))
	for i, s := range resp.Shipments {
		summaries[i] = shipmentSummaryFromProto(s)
	}

	return apiresource.NewList(summaries, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *shipmentSvcImpl) GetShipment(ctx context.Context, req *RetrieveShipmentRequest) (*apiresource.ShipmentDetail, *apierror.APIError) {
	pbReq := &pb.GetShipmentRequest{
		Id:       req.ShipmentID,
		Includes: resourcekit.FilterIncludes(ctx, shipmentIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetShipmentResponse, error) {
			return m.coreClient.GetShipment(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := shipmentDetailFromProto(resp.Shipment)
	stashShipmentDetailMeta(ctx, resp.Shipment, &result)
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
		Includes:             resourcekit.FilterIncludes(ctx, shipmentIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateShipmentResponse, error) {
			return m.coreClient.UpdateShipment(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := shipmentDetailFromProto(resp.Shipment)
	stashShipmentDetailMeta(ctx, resp.Shipment, &result)
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
		Includes:      resourcekit.FilterIncludes(ctx, shipmentIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.ship", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ShipShipmentResponse, error) {
			return m.coreClient.ShipShipment(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := shipmentDetailFromProto(resp.Shipment)
	stashShipmentDetailMeta(ctx, resp.Shipment, &result)
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

	result := shipmentDetailFromProtoFull(resp.Shipment)
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

	return estimateRateFromProto(resp), nil
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

	return rateShopFromProto(resp), nil
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

	if resp == nil {
		return apiresource.NewList[apiresource.ShipmentLine](nil, apiresource.PageInfo{}), nil
	}

	lines := make([]apiresource.ShipmentLine, len(resp.ShipmentLines))
	for i, l := range resp.ShipmentLines {
		lines[i] = shipmentLineFromProto(l)
	}

	return apiresource.NewList(lines, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
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

	result := shipmentLineFromProto(resp.ShipmentLine)
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

	result := shipmentLineFromProto(resp.ShipmentLine)
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

	result := shipmentLineFromProto(resp.ShipmentLine)
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

// shipmentDetailFromProto builds a ShipmentDetail with only non-expandable fields.
// Expandable sub-resources (sales_order, pick, customer, carrier, service_level,
// shipping_address, shipped_by, invoice, lines, shipping_cases) are left nil
// and populated later by the V2 include resolver via stashShipmentDetailMeta.
func shipmentDetailFromProto(s *pb.ShipmentInfo) apiresource.ShipmentDetail {
	if s == nil {
		return apiresource.ShipmentDetail{}
	}

	result := apiresource.ShipmentDetail{
		ID:                   s.Id,
		Object:               constants.ObjectTypeShipment,
		Number:               s.Number,
		Note:                 s.Note,
		BillOfLading:         s.BillOfLading,
		MasterTrackingNumber: s.MasterTrackingNumber,
		Status: apiresource.ShipmentStatus{
			Code: s.StatusCode,
			Name: s.StatusName,
		},
		ShippedAt: grpcutil.TimestampToTimePtr(s.ShippedAt),
		CreatedAt: grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(s.UpdatedAt),
	}

	if s.CarrierBillingType != nil && *s.CarrierBillingType != "" {
		result.Billing = &apiresource.ShipmentBilling{
			Type:    *s.CarrierBillingType,
			Account: s.CarrierBillingAccount,
			Country: s.BillingAddressCountry,
			Zip:     s.BillingAddressZip,
		}
	}

	return result
}

// stashShipmentDetailMeta stashes all expandable sub-resource data into the
// resourcekit load meta so the V2 include resolver can populate them.
func stashShipmentDetailMeta(ctx context.Context, s *pb.ShipmentInfo, d *apiresource.ShipmentDetail) {
	if s == nil {
		return
	}

	meta := resourcekit.GetLoadMeta(ctx)

	if s.SalesOrderId != "" {
		so := &apiresource.SalesOrderDetail{
			ID:         s.SalesOrderId,
			Object:     constants.ObjectTypeSalesOrder,
			Number:     s.SalesOrderNumber,
			CustomerPO: s.CustomerPoNumber,
		}
		if s.SalesOrderCreatedAt != nil {
			so.CreatedAt = s.SalesOrderCreatedAt.AsTime()
		}
		if s.SalesOrderUpdatedAt != nil {
			so.UpdatedAt = s.SalesOrderUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypeShipment, d.ID, "sales_order", so)
	}

	if s.PickId != nil && *s.PickId != "" {
		pick := &apiresource.PickDetail{
			ID:     *s.PickId,
			Object: constants.ObjectTypePick,
		}
		if s.PickNumber != nil {
			pick.Number = *s.PickNumber
		}
		if s.PickCreatedAt != nil {
			pick.CreatedAt = s.PickCreatedAt.AsTime()
		}
		if s.PickUpdatedAt != nil {
			pick.UpdatedAt = s.PickUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypeShipment, d.ID, "pick", pick)
	}

	if s.CustomerId != "" {
		customer := &apiresource.Customer{
			ID:               s.CustomerId,
			Object:           constants.ObjectTypeCustomer,
			Name:             s.CustomerName,
			Number:           s.CustomerNumber,
			EDIStatus:        constants.EDIStatusDisabled,
			RelationshipType: constants.CustomerRelationshipTypeStandalone,
		}
		if s.CustomerStatusCode != nil {
			customer.Status = constants.AccountStatusCode(*s.CustomerStatusCode)
		}
		if s.CustomerCommissionPolicy != nil {
			customer.CommissionPolicy = constants.CommissionPolicy(*s.CustomerCommissionPolicy)
		}
		if s.CustomerCreatedAt != nil {
			customer.CreatedAt = s.CustomerCreatedAt.AsTime()
		}
		if s.CustomerUpdatedAt != nil {
			customer.UpdatedAt = s.CustomerUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypeShipment, d.ID, "customer", customer)
	}

	if s.CarrierId != "" {
		carrier := &apiresource.Carrier{
			ID:     s.CarrierId,
			Object: constants.ObjectTypeCarrier,
			Name:   s.CarrierName,
		}
		if s.CarrierIsPortalEnabled != nil && *s.CarrierIsPortalEnabled {
			carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityVisible
		} else {
			carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityHidden
		}
		if s.CarrierCreatedAt != nil {
			carrier.CreatedAt = s.CarrierCreatedAt.AsTime()
		}
		if s.CarrierUpdatedAt != nil {
			carrier.UpdatedAt = s.CarrierUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypeShipment, d.ID, "carrier", carrier)
	}

	if s.ServiceLevelId != nil && *s.ServiceLevelId != "" {
		sl := &apiresource.ServiceLevel{
			ID:     *s.ServiceLevelId,
			Object: constants.ObjectTypeServiceLevel,
			Name:   derefStr(s.ServiceLevelName),
		}
		if s.ServiceLevelIsPortalEnabled != nil && *s.ServiceLevelIsPortalEnabled {
			sl.CustomerPortalVisibility = constants.CustomerPortalVisibilityVisible
		} else {
			sl.CustomerPortalVisibility = constants.CustomerPortalVisibilityHidden
		}
		if s.ServiceLevelToken != nil {
			sl.ServiceLevelToken = constants.ServiceLevelCode(*s.ServiceLevelToken)
		}
		if s.ServiceLevelCreatedAt != nil {
			sl.CreatedAt = s.ServiceLevelCreatedAt.AsTime()
		}
		if s.ServiceLevelUpdatedAt != nil {
			sl.UpdatedAt = s.ServiceLevelUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypeShipment, d.ID, "service_level", sl)
	}

	if s.ShippingAddressId != "" {
		addr := &apiresource.Address{
			ID:     s.ShippingAddressId,
			Object: constants.ObjectTypeAddress,
			Name:   derefStr(s.ShippingAddressName),
			Type:   constants.AddressTypeStandard,
		}
		if s.ShippingAddressCreatedAt != nil {
			addr.CreatedAt = s.ShippingAddressCreatedAt.AsTime()
		}
		if s.ShippingAddressUpdatedAt != nil {
			addr.UpdatedAt = s.ShippingAddressUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypeShipment, d.ID, "shipping_address", addr)
	}

	if s.ShippedById != nil && *s.ShippedById != "" {
		shippedBy := &apiresource.AccountUser{
			ID:     *s.ShippedById,
			Object: constants.ObjectTypeAccountUser,
			Name:   s.ShippedByName,
		}
		if s.ShippedByStatus != nil {
			shippedBy.Status = constants.AccountUserStatus(*s.ShippedByStatus)
		}
		if s.ShippedByCreatedAt != nil {
			shippedBy.CreatedAt = s.ShippedByCreatedAt.AsTime()
		}
		if s.ShippedByUpdatedAt != nil {
			shippedBy.UpdatedAt = s.ShippedByUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypeShipment, d.ID, "shipped_by", shippedBy)
	}

	if s.InvoiceId != nil && *s.InvoiceId != "" {
		inv := &apiresource.Invoice{
			ID:     *s.InvoiceId,
			Object: constants.ObjectTypeInvoice,
			Number: derefStr(s.InvoiceNumber),
		}
		if s.InvoiceCreatedAt != nil {
			inv.CreatedAt = s.InvoiceCreatedAt.AsTime()
		}
		if s.InvoiceUpdatedAt != nil {
			inv.UpdatedAt = s.InvoiceUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypeShipment, d.ID, "invoice", inv)
	}

	if s.Lines != nil {
		lines := make([]apiresource.ShipmentLine, len(s.Lines))
		for i, l := range s.Lines {
			lines[i] = shipmentLineFromProto(l)
		}
		meta.Set(constants.ObjectTypeShipment, d.ID, "lines", apiresource.NewList(lines, apiresource.PageInfo{}))
	}

	if s.ShippingCases != nil {
		cases := make([]apiresource.ShippingCaseDetail, len(s.ShippingCases))
		for i, c := range s.ShippingCases {
			cases[i] = shippingCaseDetailFromProto(c)
		}
		meta.Set(constants.ObjectTypeShipment, d.ID, "shipping_cases", apiresource.NewList(cases, apiresource.PageInfo{}))
	}
}

// shipmentDetailFromProtoFull builds a ShipmentDetail with ALL fields populated
// directly (no meta stashing). Used by endpoints without V2 include resolver
// (e.g. VoidShipment).
func shipmentDetailFromProtoFull(s *pb.ShipmentInfo) apiresource.ShipmentDetail {
	if s == nil {
		return apiresource.ShipmentDetail{}
	}

	result := apiresource.ShipmentDetail{
		ID:                   s.Id,
		Object:               constants.ObjectTypeShipment,
		Number:               s.Number,
		Note:                 s.Note,
		BillOfLading:         s.BillOfLading,
		MasterTrackingNumber: s.MasterTrackingNumber,
		Status: apiresource.ShipmentStatus{
			Code: s.StatusCode,
			Name: s.StatusName,
		},
		ShippedAt: grpcutil.TimestampToTimePtr(s.ShippedAt),
		CreatedAt: grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(s.UpdatedAt),
	}

	if s.SalesOrderId != "" {
		result.SalesOrder = &apiresource.SalesOrderDetail{
			ID:         s.SalesOrderId,
			Object:     constants.ObjectTypeSalesOrder,
			Number:     s.SalesOrderNumber,
			CustomerPO: s.CustomerPoNumber,
		}
		if s.SalesOrderCreatedAt != nil {
			result.SalesOrder.CreatedAt = s.SalesOrderCreatedAt.AsTime()
		}
		if s.SalesOrderUpdatedAt != nil {
			result.SalesOrder.UpdatedAt = s.SalesOrderUpdatedAt.AsTime()
		}
	}

	if s.PickId != nil && *s.PickId != "" {
		result.Pick = &apiresource.PickDetail{
			ID:     *s.PickId,
			Object: constants.ObjectTypePick,
		}
		if s.PickNumber != nil {
			result.Pick.Number = *s.PickNumber
		}
		if s.PickCreatedAt != nil {
			result.Pick.CreatedAt = s.PickCreatedAt.AsTime()
		}
		if s.PickUpdatedAt != nil {
			result.Pick.UpdatedAt = s.PickUpdatedAt.AsTime()
		}
	}

	if s.CarrierBillingType != nil && *s.CarrierBillingType != "" {
		result.Billing = &apiresource.ShipmentBilling{
			Type:    *s.CarrierBillingType,
			Account: s.CarrierBillingAccount,
			Country: s.BillingAddressCountry,
			Zip:     s.BillingAddressZip,
		}
	}

	if s.CustomerId != "" {
		result.Customer = &apiresource.Customer{
			ID:               s.CustomerId,
			Object:           constants.ObjectTypeCustomer,
			Name:             s.CustomerName,
			Number:           s.CustomerNumber,
			EDIStatus:        constants.EDIStatusDisabled,
			RelationshipType: constants.CustomerRelationshipTypeStandalone,
		}
		if s.CustomerStatusCode != nil {
			result.Customer.Status = constants.AccountStatusCode(*s.CustomerStatusCode)
		}
		if s.CustomerCommissionPolicy != nil {
			result.Customer.CommissionPolicy = constants.CommissionPolicy(*s.CustomerCommissionPolicy)
		}
		if s.CustomerCreatedAt != nil {
			result.Customer.CreatedAt = s.CustomerCreatedAt.AsTime()
		}
		if s.CustomerUpdatedAt != nil {
			result.Customer.UpdatedAt = s.CustomerUpdatedAt.AsTime()
		}
	}

	if s.CarrierId != "" {
		result.Carrier = &apiresource.Carrier{
			ID:     s.CarrierId,
			Object: constants.ObjectTypeCarrier,
			Name:   s.CarrierName,
		}
		if s.CarrierIsPortalEnabled != nil && *s.CarrierIsPortalEnabled {
			result.Carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityVisible
		} else {
			result.Carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityHidden
		}
		if s.CarrierCreatedAt != nil {
			result.Carrier.CreatedAt = s.CarrierCreatedAt.AsTime()
		}
		if s.CarrierUpdatedAt != nil {
			result.Carrier.UpdatedAt = s.CarrierUpdatedAt.AsTime()
		}
	}

	if s.ServiceLevelId != nil && *s.ServiceLevelId != "" {
		result.ServiceLevel = &apiresource.ServiceLevel{
			ID:     *s.ServiceLevelId,
			Object: constants.ObjectTypeServiceLevel,
			Name:   derefStr(s.ServiceLevelName),
		}
		if s.ServiceLevelIsPortalEnabled != nil && *s.ServiceLevelIsPortalEnabled {
			result.ServiceLevel.CustomerPortalVisibility = constants.CustomerPortalVisibilityVisible
		} else {
			result.ServiceLevel.CustomerPortalVisibility = constants.CustomerPortalVisibilityHidden
		}
		if s.ServiceLevelToken != nil {
			result.ServiceLevel.ServiceLevelToken = constants.ServiceLevelCode(*s.ServiceLevelToken)
		}
		if s.ServiceLevelCreatedAt != nil {
			result.ServiceLevel.CreatedAt = s.ServiceLevelCreatedAt.AsTime()
		}
		if s.ServiceLevelUpdatedAt != nil {
			result.ServiceLevel.UpdatedAt = s.ServiceLevelUpdatedAt.AsTime()
		}
	}

	if s.ShippingAddressId != "" {
		result.ShippingAddress = &apiresource.Address{
			ID:     s.ShippingAddressId,
			Object: constants.ObjectTypeAddress,
			Name:   derefStr(s.ShippingAddressName),
			Type:   constants.AddressTypeStandard,
		}
		if s.ShippingAddressCreatedAt != nil {
			result.ShippingAddress.CreatedAt = s.ShippingAddressCreatedAt.AsTime()
		}
		if s.ShippingAddressUpdatedAt != nil {
			result.ShippingAddress.UpdatedAt = s.ShippingAddressUpdatedAt.AsTime()
		}
	}

	if s.ShippedById != nil && *s.ShippedById != "" {
		result.ShippedBy = &apiresource.AccountUser{
			ID:     *s.ShippedById,
			Object: constants.ObjectTypeAccountUser,
			Name:   s.ShippedByName,
		}
		if s.ShippedByStatus != nil {
			result.ShippedBy.Status = constants.AccountUserStatus(*s.ShippedByStatus)
		}
		if s.ShippedByCreatedAt != nil {
			result.ShippedBy.CreatedAt = s.ShippedByCreatedAt.AsTime()
		}
		if s.ShippedByUpdatedAt != nil {
			result.ShippedBy.UpdatedAt = s.ShippedByUpdatedAt.AsTime()
		}
	}

	if s.InvoiceId != nil && *s.InvoiceId != "" {
		result.Invoice = &apiresource.Invoice{
			ID:     *s.InvoiceId,
			Object: constants.ObjectTypeInvoice,
			Number: derefStr(s.InvoiceNumber),
		}
		if s.InvoiceCreatedAt != nil {
			result.Invoice.CreatedAt = s.InvoiceCreatedAt.AsTime()
		}
		if s.InvoiceUpdatedAt != nil {
			result.Invoice.UpdatedAt = s.InvoiceUpdatedAt.AsTime()
		}
	}

	if s.Lines != nil {
		lines := make([]apiresource.ShipmentLine, len(s.Lines))
		for i, l := range s.Lines {
			lines[i] = shipmentLineFromProto(l)
		}
		result.Lines = apiresource.NewList(lines, apiresource.PageInfo{})
	}

	if s.ShippingCases != nil {
		cases := make([]apiresource.ShippingCaseDetail, len(s.ShippingCases))
		for i, c := range s.ShippingCases {
			cases[i] = shippingCaseDetailFromProto(c)
		}
		result.ShippingCases = apiresource.NewList(cases, apiresource.PageInfo{})
	}

	return result
}

func shipmentSummaryFromProto(s *pb.ShipmentSummaryInfo) apiresource.ShipmentSummary {
	if s == nil {
		return apiresource.ShipmentSummary{}
	}

	result := apiresource.ShipmentSummary{
		ID:                   s.Id,
		Object:               constants.ObjectTypeShipmentSummary,
		Number:               s.Number,
		Note:                 s.Note,
		BillOfLading:         s.BillOfLading,
		MasterTrackingNumber: s.MasterTrackingNumber,
		Status: apiresource.ShipmentStatus{
			Code: s.StatusCode,
			Name: s.StatusName,
		},
		ShippedAt: grpcutil.TimestampToTimePtr(s.ShippedAt),
		CreatedAt: grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(s.UpdatedAt),
	}

	if s.SalesOrderId != "" {
		result.SalesOrder = &apiresource.SalesOrderDetail{
			ID:        s.SalesOrderId,
			Object:    constants.ObjectTypeSalesOrder,
			Number:    s.SalesOrderNumber,
			CreatedAt: grpcutil.TimestampToTime(s.SalesOrderCreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(s.SalesOrderUpdatedAt),
		}
	}

	if s.CustomerId != "" {
		result.Customer = &apiresource.Customer{
			ID:               s.CustomerId,
			Object:           constants.ObjectTypeCustomer,
			Name:             s.CustomerName,
			Number:           s.CustomerNumber,
			EDIStatus:        constants.EDIStatusDisabled,
			RelationshipType: constants.CustomerRelationshipTypeStandalone,
			CreatedAt:        grpcutil.TimestampToTime(s.CustomerCreatedAt),
			UpdatedAt:        grpcutil.TimestampToTime(s.CustomerUpdatedAt),
		}
		if s.CustomerStatusCode != nil {
			result.Customer.Status = constants.AccountStatusCode(*s.CustomerStatusCode)
		}
		if s.CustomerCommissionPolicy != nil {
			result.Customer.CommissionPolicy = constants.CommissionPolicy(*s.CustomerCommissionPolicy)
		}
	}

	if s.CarrierId != "" {
		result.Carrier = &apiresource.Carrier{
			ID:     s.CarrierId,
			Object: constants.ObjectTypeCarrier,
			Name:   s.CarrierName,
		}
		if s.CarrierIsPortalEnabled != nil && *s.CarrierIsPortalEnabled {
			result.Carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityVisible
		} else {
			result.Carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityHidden
		}
	}

	if s.ServiceLevelId != nil && *s.ServiceLevelId != "" {
		result.ServiceLevel = &apiresource.ServiceLevel{
			ID:     *s.ServiceLevelId,
			Object: constants.ObjectTypeServiceLevel,
			Name:   derefStr(s.ServiceLevelName),
		}
		if s.ServiceLevelIsPortalEnabled != nil && *s.ServiceLevelIsPortalEnabled {
			result.ServiceLevel.CustomerPortalVisibility = constants.CustomerPortalVisibilityVisible
		} else {
			result.ServiceLevel.CustomerPortalVisibility = constants.CustomerPortalVisibilityHidden
		}
		if s.ServiceLevelToken != nil {
			result.ServiceLevel.ServiceLevelToken = constants.ServiceLevelCode(*s.ServiceLevelToken)
		}
	}

	return result
}

func shipmentLineFromProto(l *pb.ShipmentLineInfo) apiresource.ShipmentLine {
	if l == nil {
		return apiresource.ShipmentLine{}
	}

	result := apiresource.ShipmentLine{
		ID:     l.Id,
		Object: constants.ObjectTypeShipmentLine,
		Quantity: &apiresource.Quantity{
			ID:           l.QuantityId,
			Object:       constants.ObjectTypeQuantity,
			Value:        l.QuantityValue,
			DisplayValue: fmt.Sprintf("%s %s", l.QuantityValue, l.QuantityUnitAbbreviation),
		},
		CreatedAt: grpcutil.TimestampToTime(l.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(l.UpdatedAt),
	}

	return result
}

func shippingCaseDetailFromProto(c *pb.ShippingCaseDetailInfo) apiresource.ShippingCaseDetail {
	if c == nil {
		return apiresource.ShippingCaseDetail{}
	}

	result := apiresource.ShippingCaseDetail{
		ID:                  c.Id,
		Object:              constants.ObjectTypeShippingCase,
		Number:              c.Number,
		SSCC:                c.Sscc,
		TrackingNumber:      c.TrackingNumber,
		ShippoTransactionID: c.ShippoTransactionId,
		ShippingLabelURL:    c.ShippingLabelUrl,
		ShippedAt:           grpcutil.TimestampToTimePtr(c.ShippedAt),
		CreatedAt:           grpcutil.TimestampToTime(c.CreatedAt),
		UpdatedAt:           grpcutil.TimestampToTime(c.UpdatedAt),
	}

	if c.FreightAmountId != "" {
		result.FreightAmount = &apiresource.Quantity{
			ID:           c.FreightAmountId,
			Object:       constants.ObjectTypeQuantity,
			Value:        c.FreightAmountValue,
			DisplayValue: fmt.Sprintf("%s %s", c.FreightAmountValue, c.FreightAmountUnitAbbreviation),
		}
	}

	if c.FreightWeightId != "" {
		result.FreightWeight = &apiresource.Quantity{
			ID:           c.FreightWeightId,
			Object:       constants.ObjectTypeQuantity,
			Value:        c.FreightWeightValue,
			DisplayValue: fmt.Sprintf("%s %s", c.FreightWeightValue, c.FreightWeightUnitAbbreviation),
		}
	}

	if c.CarrierId != "" {
		result.Carrier = &apiresource.Carrier{
			ID:     c.CarrierId,
			Object: constants.ObjectTypeCarrier,
			Name:   c.CarrierName,
		}
		if c.CarrierIsPortalEnabled != nil && *c.CarrierIsPortalEnabled {
			result.Carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityVisible
		} else {
			result.Carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityHidden
		}
		if c.CarrierCreatedAt != nil {
			result.Carrier.CreatedAt = c.CarrierCreatedAt.AsTime()
		}
		if c.CarrierUpdatedAt != nil {
			result.Carrier.UpdatedAt = c.CarrierUpdatedAt.AsTime()
		}
	}

	return result
}

func estimateRateFromProto(resp *pb.EstimateRateResponse) *apiresource.EstimateRateResult {
	if resp == nil {
		return nil
	}
	return &apiresource.EstimateRateResult{
		Object: constants.ObjectTypeEstimateRateResult,
		Rate:   resp.Rate,
	}
}

func rateShopFromProto(resp *pb.RateShopResponse) *apiresource.RateShopResult {
	if resp == nil {
		return nil
	}

	options := make([]apiresource.RateShopOption, len(resp.Options))
	for i, opt := range resp.Options {
		options[i] = apiresource.RateShopOption{
			Object: constants.ObjectTypeRateShopOption,
			Carrier: &apiresource.Carrier{
				ID:     opt.CarrierId,
				Object: constants.ObjectTypeCarrier,
				Name:   opt.CarrierName,
			},
			ServiceLevel: &apiresource.ServiceLevel{
				ID:     opt.ServiceLevelId,
				Object: constants.ObjectTypeServiceLevel,
				Name:   opt.ServiceLevelName,
			},
			Rate:          opt.Rate,
			EstimatedDays: opt.EstimatedDays,
		}
	}

	return &apiresource.RateShopResult{
		Object:        constants.ObjectTypeRateShopResult,
		Options:       apiresource.NewList(options, apiresource.PageInfo{}),
		ExemptionType: resp.ExemptionType,
		FlatRate:      resp.FlatRate,
	}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
