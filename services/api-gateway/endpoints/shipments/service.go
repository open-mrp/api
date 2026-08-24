package shipmentep

import (
	"context"
	"fmt"
	"strconv"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/ptrutil"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ShipmentSvc interface {
	ListShipments(ctx context.Context, req *ListShipmentsRequest) (*apiresource.List[apiresource.Shipment], *apierror.APIError)
	GetShipment(ctx context.Context, req *RetrieveShipmentRequest) (*apiresource.Shipment, *apierror.APIError)
	UpdateShipment(ctx context.Context, req *UpdateShipmentRequest) (*apiresource.Shipment, *apierror.APIError)
	AdminUpdateShipmentTracking(ctx context.Context, req *AdminUpdateShipmentTrackingRequest) (*apiresource.Shipment, *apierror.APIError)
	DeleteShipment(ctx context.Context, req *DeleteShipmentRequest) (*apiresource.EmptyResource, *apierror.APIError)
	ShipShipment(ctx context.Context, req *ShipShipmentRequest) (*apiresource.Shipment, *apierror.APIError)
	VoidShipment(ctx context.Context, req *VoidShipmentRequest) (*apiresource.Shipment, *apierror.APIError)
	EstimateRate(ctx context.Context, req *EstimateRateRequest) (*apiresource.EstimateRateResult, *apierror.APIError)
	RateShop(ctx context.Context, req *RateShopRequest) (*apiresource.RateShopResult, *apierror.APIError)
	ListShipmentLines(ctx context.Context, req *ListShipmentLinesRequest) (*apiresource.List[apiresource.ShipmentLine], *apierror.APIError)
	GetShipmentLine(ctx context.Context, req *RetrieveShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError)
	CreateShipmentLine(ctx context.Context, req *CreateShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError)
	UpdateShipmentLine(ctx context.Context, req *UpdateShipmentLineRequest) (*apiresource.ShipmentLine, *apierror.APIError)
	DeleteShipmentLine(ctx context.Context, req *DeleteShipmentLineRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type ShipmentSvcConfig struct {
	// CoreClient (required) is the core-service shipping gRPC client.
	CoreClient pb.CoreShippingServiceClient
}

type shipmentSvcImpl struct {
	coreClient pb.CoreShippingServiceClient
}

var shipmentSvcTracer = tracing.GetTracer("api-gateway.endpoints.shipments.service")

var shipmentIncludes = []string{"lines", "shipping_cases", "sales_order", "customer", "freight", "shipping_address", "shipped_by", "invoice", "pick"}

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

func (m *shipmentSvcImpl) ListShipments(ctx context.Context, req *ListShipmentsRequest) (*apiresource.List[apiresource.Shipment], *apierror.APIError) {
	pbReq := &pb.ListShipmentsRequest{
		Cursor:           req.Cursor,
		Limit:            req.Limit,
		Query:            req.Query,
		Status:           req.Status.StringPtr(),
		ItemIds:          req.ItemIDs,
		CustomerIds:      req.CustomerIDs,
		ProductLineIds:   req.ProductLineIDs,
		CustomerGroupIds: req.CustomerGroupIDs,
		SalesRepIds:      req.SalesRepIDs,
		StartDate:        req.StartDate,
		EndDate:          req.EndDate,
		// Ask the backend to expand lines when requested (other includes are
		// resolved gateway-side from stashed FK ids).
		Includes: resourcekit.FilterIncludes(ctx, "lines"),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListShipmentsResponse, error) {
			return m.coreClient.ListShipments(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	if resp == nil {
		return apiresource.NewList[apiresource.Shipment](nil, apiresource.PageInfo{}), nil
	}

	shipments := make([]apiresource.Shipment, len(resp.Shipments))
	for i, s := range resp.Shipments {
		shipments[i] = shipmentFromProto(s)
		stashShipmentMeta(ctx, s, &shipments[i])
	}

	return apiresource.NewList(shipments, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *shipmentSvcImpl) GetShipment(ctx context.Context, req *RetrieveShipmentRequest) (*apiresource.Shipment, *apierror.APIError) {
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

	result := shipmentFromProto(resp.Shipment)
	stashShipmentMeta(ctx, resp.Shipment, &result)
	return &result, nil
}

func (m *shipmentSvcImpl) UpdateShipment(ctx context.Context, req *UpdateShipmentRequest) (*apiresource.Shipment, *apierror.APIError) {
	pbReq := &pb.UpdateShipmentRequest{
		Id:                   req.ShipmentID,
		Note:                 req.Note.Ptr(),
		Number:               req.Number.Ptr(),
		MasterTrackingNumber: req.MasterTrackingNumber.Ptr(),
		CarrierId:            req.CarrierID.Ptr(),
		ServiceLevelId:       field.StringClearableToProto(req.ServiceLevelID),
		Includes:             resourcekit.FilterIncludes(ctx, shipmentIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateShipmentResponse, error) {
			return m.coreClient.UpdateShipment(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := shipmentFromProto(resp.Shipment)
	stashShipmentMeta(ctx, resp.Shipment, &result)
	return &result, nil
}

func (m *shipmentSvcImpl) AdminUpdateShipmentTracking(ctx context.Context, req *AdminUpdateShipmentTrackingRequest) (*apiresource.Shipment, *apierror.APIError) {
	pbReq := &pb.AdminUpdateShipmentTrackingRequest{
		Id:                   req.ShipmentID,
		MasterTrackingNumber: req.MasterTrackingNumber.Ptr(),
		CarrierId:            req.CarrierID.Ptr(),
		ServiceLevelId:       field.StringClearableToProto(req.ServiceLevelID),
		Includes:             resourcekit.FilterIncludes(ctx, shipmentIncludes...),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.admin_update_tracking", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.AdminUpdateShipmentTrackingResponse, error) {
			return m.coreClient.AdminUpdateShipmentTracking(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := shipmentFromProto(resp.Shipment)
	stashShipmentMeta(ctx, resp.Shipment, &result)
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

func (m *shipmentSvcImpl) ShipShipment(ctx context.Context, req *ShipShipmentRequest) (*apiresource.Shipment, *apierror.APIError) {
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

	result := shipmentFromProto(resp.Shipment)
	stashShipmentMeta(ctx, resp.Shipment, &result)
	return &result, nil
}

func (m *shipmentSvcImpl) VoidShipment(ctx context.Context, req *VoidShipmentRequest) (*apiresource.Shipment, *apierror.APIError) {
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

	// VoidShipment has no V2 include resolver, so expandables are left nil.
	result := shipmentFromProto(resp.Shipment)
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
		CustomerId:     req.CustomerID.Ptr(),
		From:           addressInputToProto(req.FromAddress),
		To:             addressInputToProto(req.ToAddress),
		Parcels:        parcels,
		OrderTotal:     req.OrderTotal.Ptr(),
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

	// Origin is optional: when the caller omits it, core resolves the account's
	// ship-from origin server-side, so leave From nil here.
	var fromProto *pb.AddressInput
	if fromAddr, ok := req.FromAddress.Value(); ok {
		fromProto = addressInputToProto(fromAddr)
	}

	pbReq := &pb.RateShopRequest{
		ProductLineIds: req.ProductLineIDs,
		CustomerId:     req.CustomerID.Ptr(),
		From:           fromProto,
		To:             addressInputToProto(req.ToAddress),
		Parcels:        parcels,
		OrderTotal:     req.OrderTotal.Ptr(),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, shipmentSvcTracer, "service.shipments.rate_shop", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.RateShopResponse, error) {
			return m.coreClient.RateShop(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	// The core RateShop RPC only echoes back carrier/service-level ids and names, so batch-hydrate the full resources here — otherwise fields like service_level_token and customer_portal_visibility come back empty.
	carriers, serviceLevels, apiErr := m.loadRateShopResources(ctx, resp)
	if apiErr != nil {
		return nil, apiErr
	}

	return rateShopFromProto(resp, carriers, serviceLevels), nil
}

// loadRateShopResources batch-loads the full Carrier and ServiceLevel resources referenced by a rate-shop response, keyed by id, via the shared resource loaders.
func (m *shipmentSvcImpl) loadRateShopResources(ctx context.Context, resp *pb.RateShopResponse) (map[string]*apiresource.Carrier, map[string]*apiresource.ServiceLevel, *apierror.APIError) {
	if resp == nil || len(resp.Options) == 0 {
		return nil, nil, nil
	}

	carrierIDs := uniqueStrings(resp.Options, func(o *pb.RateShopOptionInfo) string { return o.CarrierId })
	serviceLevelIDs := uniqueStrings(resp.Options, func(o *pb.RateShopOptionInfo) string { return o.ServiceLevelId })

	carrierMap, apiErr := resourceloaders.LoadCarriers(ctx, carrierIDs)
	if apiErr != nil {
		return nil, nil, apiErr
	}
	serviceLevelMap, apiErr := resourceloaders.LoadServiceLevels(ctx, serviceLevelIDs)
	if apiErr != nil {
		return nil, nil, apiErr
	}

	carriers := make(map[string]*apiresource.Carrier, len(carrierMap))
	for id, v := range carrierMap {
		if c, ok := v.(*apiresource.Carrier); ok {
			carriers[id] = c
		}
	}
	serviceLevels := make(map[string]*apiresource.ServiceLevel, len(serviceLevelMap))
	for id, v := range serviceLevelMap {
		if sl, ok := v.(*apiresource.ServiceLevel); ok {
			serviceLevels[id] = sl
		}
	}
	return carriers, serviceLevels, nil
}

// uniqueStrings collects the distinct non-empty keys returned by keyOf across options, preserving first-seen order.
func uniqueStrings(options []*pb.RateShopOptionInfo, keyOf func(*pb.RateShopOptionInfo) string) []string {
	seen := make(map[string]bool, len(options))
	out := make([]string, 0, len(options))
	for _, o := range options {
		k := keyOf(o)
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, k)
	}
	return out
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
		QuantityValue:  req.QuantityValue.Ptr(),
		QuantityUnitId: req.QuantityUnitID.Ptr(),
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
		Street1: ptrutil.Deref(a.StreetLine1.Ptr()),
		Street2: a.StreetLine2.Ptr(),
		City:    ptrutil.Deref(a.Locality.Ptr()),
		State:   ptrutil.Deref(a.State.Ptr()),
		Zip:     ptrutil.Deref(a.PostalCode.Ptr()),
		Country: a.Country,
		Phone:   a.Phone.Ptr(),
		Email:   a.Email.Ptr(),
	}
}

// shipmentFromProto builds a Shipment with only base, non-expandable fields.
// Expandable sub-resources (sales_order, customer, freight, shipping_address,
// shipped_by, invoice, pick, lines, shipping_cases) are left nil and populated
// later by the V2 include resolver via stashShipmentMeta.
func shipmentFromProto(s *pb.ShipmentInfo) apiresource.Shipment {
	if s == nil {
		return apiresource.Shipment{}
	}

	return apiresource.Shipment{
		ID:                   s.Id,
		Object:               constants.ObjectTypeShipment,
		Number:               s.Number,
		Note:                 s.Note,
		BillOfLading:         s.BillOfLading,
		MasterTrackingNumber: s.MasterTrackingNumber,
		Status:               constants.ShipmentStatus(s.StatusCode),
		ShippedAt:            grpcutil.TimestampToTimePtr(s.ShippedAt),
		Priority:             constants.PriorityCode(s.PriorityCode),
		CaseCount:            int32(s.CaseCount), // #nosec G115 - COUNT(*) of shipping_case rows for one shipment
		IsReadyToShip:        s.IsReadyToShip,
		CreatedAt:            grpcutil.TimestampToTime(s.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(s.UpdatedAt),
	}
}

// stashShipmentMeta stashes all expandable sub-resource data into the
// resourcekit load meta so the V2 include resolver can populate them. Document-
// level cross-references (sales_order, pick, invoice) and loader-backed
// references (customer, shipping_address, shipped_by) are stashed as FK ids only.
func stashShipmentMeta(ctx context.Context, s *pb.ShipmentInfo, d *apiresource.Shipment) {
	if s == nil {
		return
	}

	meta := resourcekit.GetLoadMeta(ctx)

	// Document-level cross-references: stash only the FK id so the loader
	// fetches the real document on ?include=. Never fabricate.
	if s.SalesOrderId != "" {
		meta.Set(constants.ObjectTypeShipment, d.ID, "sales_order_id", s.SalesOrderId)
	}
	if s.PickId != nil && *s.PickId != "" {
		meta.Set(constants.ObjectTypeShipment, d.ID, "pick_id", *s.PickId)
	}
	if s.InvoiceId != nil && *s.InvoiceId != "" {
		meta.Set(constants.ObjectTypeShipment, d.ID, "invoice_id", *s.InvoiceId)
	}

	// Loader-backed references: stash the FK ids so their real resources load on
	// ?include=. Never fabricate.
	if s.CustomerId != "" {
		meta.Set(constants.ObjectTypeShipment, d.ID, "customer_id", s.CustomerId)
	}
	// Shipping address belongs to the customer account (cross-account) and is not
	// resolvable via the account-scoped loader, so it is carried inline from the
	// data the shipment query already joined (mirrors SalesOrder addresses).
	if s.ShippingAddressId != "" {
		meta.Set(constants.ObjectTypeShipment, d.ID, "shipping_address", buildShipmentShippingAddress(s))
	}
	if s.ShippedById != nil && *s.ShippedById != "" {
		meta.Set(constants.ObjectTypeShipment, d.ID, "shipped_by_id", *s.ShippedById)
	}

	// Freight (carrier selection + freight billing). Expanded as a whole via
	// include[]=freight; carries the full carrier and service level inline. The
	// proto's billing country/zip have no Freight home and are covered by the
	// shipping_address, so they are dropped here.
	meta.Set(constants.ObjectTypeShipment, d.ID, "freight", shipmentFreightFromProto(s))

	if s.Lines != nil {
		lines := make([]apiresource.ShipmentLine, len(s.Lines))
		for i, l := range s.Lines {
			lines[i] = shipmentLineFromProto(l)
			stashShipmentLineMeta(ctx, &lines[i], l)
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

// buildShipmentShippingAddress builds the shipping Address inline from the data
// the shipment query joined. The address belongs to the customer account and is
// cross-account, so it cannot be resolved by the account-scoped loader.
func buildShipmentShippingAddress(s *pb.ShipmentInfo) *apiresource.Address {
	addr := &apiresource.Address{
		ID:        s.ShippingAddressId,
		Object:    constants.ObjectTypeAddress,
		Phone:     s.ShippingAddressPhone,
		Email:     s.ShippingAddressEmail,
		Type:      constants.AddressTypeStandard,
		CreatedAt: grpcutil.TimestampToTime(s.ShippingAddressCreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(s.ShippingAddressUpdatedAt),
	}
	if s.ShippingAddressName != nil {
		addr.Name = *s.ShippingAddressName
	}
	if s.ShippingAddressIsDropShip != nil && *s.ShippingAddressIsDropShip {
		addr.Type = constants.AddressTypeDropShip
	}

	country := ""
	if s.ShippingAddressCountry != nil {
		country = *s.ShippingAddressCountry
	}
	geo := &apiresource.Geolocation{
		Object:      constants.ObjectTypeGeolocation,
		StreetLine1: s.ShippingAddressStreetLine_1,
		StreetLine2: s.ShippingAddressStreetLine_2,
		Locality:    s.ShippingAddressLocality,
		State:       s.ShippingAddressState,
		PostalCode:  s.ShippingAddressPostalCode,
		Country:     country,
	}
	if s.ShippingAddressGeolocationId != nil {
		geo.ID = *s.ShippingAddressGeolocationId
	}
	addr.Geolocation = geo

	return addr
}

// shipmentFreightFromProto builds the Freight sub-resource (carrier selection +
// freight billing) inline from a ShipmentInfo.
func shipmentFreightFromProto(s *pb.ShipmentInfo) *apiresource.Freight {
	freight := &apiresource.Freight{Object: constants.ObjectTypeFreight}
	if s.CarrierBillingType != nil {
		bt := constants.CarrierBillingType(*s.CarrierBillingType)
		freight.BillingType = &bt
	}
	freight.BillingAccountNumber = s.CarrierBillingAccount

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
		freight.Carrier = carrier
	}

	if s.ServiceLevelId != nil && *s.ServiceLevelId != "" {
		sl := &apiresource.ServiceLevel{
			ID:     *s.ServiceLevelId,
			Object: constants.ObjectTypeServiceLevel,
			Name:   ptrutil.Deref(s.ServiceLevelName),
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
		freight.ServiceLevel = sl
	}

	return freight
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

	// Item carried inline (the order line's item) so lines.item.id resolves.
	if l.OrderLineItemId != nil && *l.OrderLineItemId != "" {
		item := &apiresource.Item{ID: *l.OrderLineItemId, Object: constants.ObjectTypeItem}
		if l.OrderLineSku != "" {
			item.SKU = l.OrderLineSku
		}
		result.Item = item
	}

	return result
}

// Stashes the order line a shipment line fulfils, plus that line's product FK, so the resolver can
// serve lines.sales_order_line and its product on ?include=.
func stashShipmentLineMeta(ctx context.Context, d *apiresource.ShipmentLine, l *pb.ShipmentLineInfo) {
	meta := resourcekit.GetLoadMeta(ctx)
	line := buildSalesOrderLineForShipmentLine(l)
	meta.Set(constants.ObjectTypeShipmentLine, d.ID, "sales_order_line", line)

	// Keyed by the order line, not the shipment line, because the product loader runs against the
	// sales_order_line resource once the resolver recurses into it.
	if l.OrderLineProductId != nil && *l.OrderLineProductId != "" {
		meta.Set(constants.ObjectTypeSalesOrderLine, line.ID, "product_id", *l.OrderLineProductId)
	}
}

// Builds the sales order line reference a shipment line fulfils, from the fields the parent proto
// carries — there is no standalone sales-order-line loader to resolve it against.
func buildSalesOrderLineForShipmentLine(l *pb.ShipmentLineInfo) *apiresource.SalesOrderLine {
	now := grpcutil.TimestampToTime(l.CreatedAt)

	return &apiresource.SalesOrderLine{
		ID:                 l.SalesOrderLineId,
		Object:             constants.ObjectTypeSalesOrderLine,
		LineItemNumber:     l.OrderLineItemNumber,
		ProductSKU:         l.OrderLineSku,
		ProductDescription: l.OrderLineDescription,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
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

func rateShopFromProto(resp *pb.RateShopResponse, carriers map[string]*apiresource.Carrier, serviceLevels map[string]*apiresource.ServiceLevel) *apiresource.RateShopResult {
	if resp == nil {
		return nil
	}

	options := make([]apiresource.RateShopOption, len(resp.Options))
	for i, opt := range resp.Options {
		carrier := carriers[opt.CarrierId]
		if carrier == nil {
			carrier = &apiresource.Carrier{
				ID:     opt.CarrierId,
				Object: constants.ObjectTypeCarrier,
				Name:   opt.CarrierName,
			}
		}
		serviceLevel := serviceLevels[opt.ServiceLevelId]
		if serviceLevel == nil {
			serviceLevel = &apiresource.ServiceLevel{
				ID:     opt.ServiceLevelId,
				Object: constants.ObjectTypeServiceLevel,
				Name:   opt.ServiceLevelName,
			}
		}
		options[i] = apiresource.RateShopOption{
			Object:        constants.ObjectTypeRateShopOption,
			Carrier:       carrier,
			ServiceLevel:  serviceLevel,
			Rate:          opt.Rate,
			EstimatedDays: opt.EstimatedDays,
		}
	}

	return &apiresource.RateShopResult{
		Object:        constants.ObjectTypeRateShopResult,
		Options:       apiresource.NewList(options, apiresource.PageInfo{}),
		ExemptionType: constants.EnumPtr[constants.FreightExemptionType](resp.ExemptionType),
		FlatRate:      resp.FlatRate,
	}
}
