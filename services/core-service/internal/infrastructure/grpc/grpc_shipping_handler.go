package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type shippingGRPCHandler struct {
	pb.UnimplementedCoreShippingServiceServer

	shipmentSvc     domain.ShipmentSvc
	shipmentLineSvc domain.ShipmentLineSvc
}

func shipmentToProto(s *domain.Shipment) *pb.ShipmentInfo {
	if s == nil {
		return nil
	}

	info := &pb.ShipmentInfo{
		Id:                s.ID,
		Number:            s.Number,
		StatusCode:        s.StatusCode,
		StatusName:        s.StatusName,
		SalesOrderId:      s.SalesOrderID,
		SalesOrderNumber:  s.SalesOrderNumber,
		CustomerId:        s.CustomerID,
		CustomerName:      s.CustomerName,
		CustomerNumber:    s.CustomerNumber,
		CarrierId:         s.CarrierID,
		CarrierName:       s.CarrierName,
		ShippingAddressId: s.ShippingAddressID,
		PriorityCode:      s.PriorityCode,
		CaseCount:         s.CaseCount,
		IsReadyToShip:     s.IsReadyToShip,
		AccountId:         s.AccountID,
		CreatedAt:         timestamppb.New(s.CreatedAt),
		UpdatedAt:         timestamppb.New(s.UpdatedAt),
	}

	if s.Note != nil {
		info.Note = s.Note
	}
	if s.BillOfLading != nil {
		info.BillOfLading = s.BillOfLading
	}
	if s.MasterTrackingNumber != nil {
		info.MasterTrackingNumber = s.MasterTrackingNumber
	}
	if s.ShippedAt != nil {
		info.ShippedAt = timestamppb.New(*s.ShippedAt)
	}
	if s.CarrierIsPortalEnabled != nil {
		info.CarrierIsPortalEnabled = s.CarrierIsPortalEnabled
	}
	if s.CarrierCode != nil {
		info.CarrierCode = s.CarrierCode
	}
	if s.CarrierCreatedAt != nil {
		info.CarrierCreatedAt = timestamppb.New(*s.CarrierCreatedAt)
	}
	if s.CarrierUpdatedAt != nil {
		info.CarrierUpdatedAt = timestamppb.New(*s.CarrierUpdatedAt)
	}
	if s.ServiceLevelID != nil {
		info.ServiceLevelId = s.ServiceLevelID
	}
	if s.ServiceLevelName != nil {
		info.ServiceLevelName = s.ServiceLevelName
	}
	if s.ServiceLevelIsPortalEnabled != nil {
		info.ServiceLevelIsPortalEnabled = s.ServiceLevelIsPortalEnabled
	}
	if s.ShippingAddressName != nil {
		info.ShippingAddressName = s.ShippingAddressName
	}
	info.ShippingAddressPhone = s.ShippingAddressPhone
	info.ShippingAddressEmail = s.ShippingAddressEmail
	info.ShippingAddressIsDropShip = s.ShippingAddressIsDropShip
	info.ShippingAddressGeolocationId = s.ShippingAddressGeolocationID
	info.ShippingAddressStreetLine_1 = s.ShippingAddressStreetLine1
	info.ShippingAddressStreetLine_2 = s.ShippingAddressStreetLine2
	info.ShippingAddressLocality = s.ShippingAddressLocality
	info.ShippingAddressState = s.ShippingAddressState
	info.ShippingAddressPostalCode = s.ShippingAddressPostalCode
	info.ShippingAddressCountry = s.ShippingAddressCountry
	if s.ShippedByID != nil {
		info.ShippedById = s.ShippedByID
	}
	if s.ShippedByName != nil {
		info.ShippedByName = s.ShippedByName
	}
	if s.InvoiceID != nil {
		info.InvoiceId = s.InvoiceID
	}
	if s.InvoiceNumber != nil {
		info.InvoiceNumber = s.InvoiceNumber
	}
	if s.CustomerPONumber != nil {
		info.CustomerPoNumber = s.CustomerPONumber
	}
	if s.CarrierBillingType != nil {
		info.CarrierBillingType = s.CarrierBillingType
	}
	if s.CarrierBillingAccount != nil {
		info.CarrierBillingAccount = s.CarrierBillingAccount
	}
	if s.PickID != nil {
		info.PickId = s.PickID
	}
	if s.PickNumber != nil {
		info.PickNumber = s.PickNumber
	}
	if s.ServiceLevelToken != nil {
		info.ServiceLevelToken = s.ServiceLevelToken
	}
	if s.ServiceLevelCreatedAt != nil {
		info.ServiceLevelCreatedAt = timestamppb.New(*s.ServiceLevelCreatedAt)
	}
	if s.ServiceLevelUpdatedAt != nil {
		info.ServiceLevelUpdatedAt = timestamppb.New(*s.ServiceLevelUpdatedAt)
	}
	if s.CustomerStatusCode != nil {
		info.CustomerStatusCode = s.CustomerStatusCode
	}
	if s.CustomerCommissionPolicy != nil {
		info.CustomerCommissionPolicy = s.CustomerCommissionPolicy
	}
	if s.BillingAddressCountry != nil {
		info.BillingAddressCountry = s.BillingAddressCountry
	}
	if s.BillingAddressZip != nil {
		info.BillingAddressZip = s.BillingAddressZip
	}
	info.CustomerCreatedAt = timestamppb.New(s.CustomerCreatedAt)
	info.CustomerUpdatedAt = timestamppb.New(s.CustomerUpdatedAt)
	info.SalesOrderCreatedAt = timestamppb.New(s.SalesOrderCreatedAt)
	info.SalesOrderUpdatedAt = timestamppb.New(s.SalesOrderUpdatedAt)
	if s.ShippingAddressCreatedAt != nil {
		info.ShippingAddressCreatedAt = timestamppb.New(*s.ShippingAddressCreatedAt)
	}
	if s.ShippingAddressUpdatedAt != nil {
		info.ShippingAddressUpdatedAt = timestamppb.New(*s.ShippingAddressUpdatedAt)
	}
	if s.ShippedByStatusCode != nil {
		info.ShippedByStatus = s.ShippedByStatusCode
	}
	if s.ShippedByCreatedAt != nil {
		info.ShippedByCreatedAt = timestamppb.New(*s.ShippedByCreatedAt)
	}
	if s.ShippedByUpdatedAt != nil {
		info.ShippedByUpdatedAt = timestamppb.New(*s.ShippedByUpdatedAt)
	}
	if s.InvoiceCreatedAt != nil {
		info.InvoiceCreatedAt = timestamppb.New(*s.InvoiceCreatedAt)
	}
	if s.InvoiceUpdatedAt != nil {
		info.InvoiceUpdatedAt = timestamppb.New(*s.InvoiceUpdatedAt)
	}
	if s.PickCreatedAt != nil {
		info.PickCreatedAt = timestamppb.New(*s.PickCreatedAt)
	}
	if s.PickUpdatedAt != nil {
		info.PickUpdatedAt = timestamppb.New(*s.PickUpdatedAt)
	}

	if s.Lines != nil {
		lines := make([]*pb.ShipmentLineInfo, len(s.Lines))
		for i, l := range s.Lines {
			lines[i] = shipmentLineToProto(l)
		}
		info.Lines = lines
	}

	if s.ShippingCases != nil {
		cases := make([]*pb.ShippingCaseDetailInfo, len(s.ShippingCases))
		for i, c := range s.ShippingCases {
			cases[i] = shippingCaseDetailToProto(c)
		}
		info.ShippingCases = cases
	}

	return info
}

func shipmentLineToProto(l *domain.ShipmentLine) *pb.ShipmentLineInfo {
	if l == nil {
		return nil
	}

	info := &pb.ShipmentLineInfo{
		Id:                       l.ID,
		ShipmentId:               l.ShipmentID,
		SalesOrderLineId:         l.SalesOrderLineID,
		OrderLineSku:             l.OrderLineSKU,
		OrderLineItemNumber:      l.OrderLineItemNumber,
		QuantityId:               l.QuantityID,
		QuantityValue:            l.QuantityValue,
		QuantityUnitId:           l.QuantityUnitID,
		QuantityUnitName:         l.QuantityUnitName,
		QuantityUnitAbbreviation: l.QuantityUnitAbbreviation,
		QuantityUnitType:         l.QuantityUnitType,
		CreatedAt:                timestamppb.New(l.CreatedAt),
		UpdatedAt:                timestamppb.New(l.UpdatedAt),
	}

	if l.OrderLineDesc != nil {
		info.OrderLineDescription = l.OrderLineDesc
	}
	if l.OrderLineItemID != nil {
		info.OrderLineItemId = l.OrderLineItemID
	}
	if l.OrderLineProductID != nil {
		info.OrderLineProductId = l.OrderLineProductID
	}

	return info
}

func shippingCaseDetailToProto(c *domain.ShippingCase) *pb.ShippingCaseDetailInfo {
	if c == nil {
		return nil
	}

	info := &pb.ShippingCaseDetailInfo{
		Id:                            c.ID,
		Number:                        c.Number,
		FreightAmountId:               c.FreightAmountID,
		FreightAmountValue:            c.FreightAmountValue,
		FreightAmountUnitId:           c.FreightAmountUnitID,
		FreightAmountUnitName:         c.FreightAmountUnitName,
		FreightAmountUnitAbbreviation: c.FreightAmountUnitAbbreviation,
		FreightWeightId:               c.FreightWeightID,
		FreightWeightValue:            c.FreightWeightValue,
		FreightWeightUnitId:           c.FreightWeightUnitID,
		FreightWeightUnitName:         c.FreightWeightUnitName,
		FreightWeightUnitAbbreviation: c.FreightWeightUnitAbbreviation,
		ShipmentId:                    c.ShipmentID,
		CarrierId:                     c.CarrierID,
		CarrierName:                   c.CarrierName,
		CreatedAt:                     timestamppb.New(c.CreatedAt),
		UpdatedAt:                     timestamppb.New(c.UpdatedAt),
	}

	if c.SSCC != nil {
		info.Sscc = c.SSCC
	}
	if c.TrackingNumber != nil {
		info.TrackingNumber = c.TrackingNumber
	}
	if c.ShippoTransactionID != nil {
		info.ShippoTransactionId = c.ShippoTransactionID
	}
	if c.ShippingLabelURL != nil {
		info.ShippingLabelUrl = c.ShippingLabelURL
	}
	if c.ShippedAt != nil {
		info.ShippedAt = timestamppb.New(*c.ShippedAt)
	}
	portalEnabled := c.CarrierIsPortalEnabled
	info.CarrierIsPortalEnabled = &portalEnabled
	if !c.CarrierCreatedAt.IsZero() {
		info.CarrierCreatedAt = timestamppb.New(c.CarrierCreatedAt)
	}
	if !c.CarrierUpdatedAt.IsZero() {
		info.CarrierUpdatedAt = timestamppb.New(c.CarrierUpdatedAt)
	}

	return info
}

func addressInputToDomain(a *pb.AddressInput) domain.ShippingAddress {
	if a == nil {
		return domain.ShippingAddress{}
	}

	addr := domain.ShippingAddress{
		Name:    a.Name,
		Street1: a.Street1,
		City:    a.City,
		State:   a.State,
		Zip:     a.Zip,
		Country: a.Country,
	}

	if a.Company != nil {
		addr.Company = a.Company
	}
	if a.Street2 != nil {
		addr.Street2 = a.Street2
	}
	if a.Phone != nil {
		addr.Phone = a.Phone
	}
	if a.Email != nil {
		addr.Email = a.Email
	}

	return addr
}

func parcelInfoToDomain(p *pb.ParcelInfo) domain.Parcel {
	if p == nil {
		return domain.Parcel{}
	}

	return domain.Parcel{
		Weight: p.Weight,
		Length: p.Length,
		Width:  p.Width,
		Height: p.Height,
	}
}

func rateShopOptionToProto(o *domain.RateShopOption) *pb.RateShopOptionInfo {
	if o == nil {
		return nil
	}

	info := &pb.RateShopOptionInfo{
		CarrierId:        o.CarrierID,
		CarrierName:      o.CarrierName,
		ServiceLevelId:   o.ServiceLevelID,
		ServiceLevelName: o.ServiceLevelName,
		Rate:             o.Rate,
	}

	if o.EstimatedDays != nil {
		info.EstimatedDays = o.EstimatedDays
	}

	return info
}

// ListShipments returns a paginated list of shipments.
func (h *shippingGRPCHandler) ListShipments(ctx context.Context, req *pb.ListShipmentsRequest) (*pb.ListShipmentsResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListShipmentsParams{
		Limit:    req.Limit,
		Includes: req.Includes,
	}

	if req.Cursor != nil {
		params.Cursor = req.Cursor
	}
	if req.Query != nil {
		params.Query = req.Query
	}
	if req.Status != nil {
		params.Status = req.Status
	}
	if len(req.ItemIds) > 0 {
		params.ItemIDs = req.ItemIds
	}
	if len(req.CustomerIds) > 0 {
		params.CustomerIDs = req.CustomerIds
	}
	if len(req.ProductLineIds) > 0 {
		params.ProductLineIDs = req.ProductLineIds
	}
	if len(req.CustomerGroupIds) > 0 {
		params.CustomerGroupIDs = req.CustomerGroupIds
	}
	if len(req.SalesRepIds) > 0 {
		params.SalesRepIDs = req.SalesRepIds
	}
	if req.StartDate != nil {
		params.StartDate = req.StartDate
	}
	if req.EndDate != nil {
		params.EndDate = req.EndDate
	}

	result, apiErr := h.shipmentSvc.ListShipments(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	shipments := make([]*pb.ShipmentInfo, len(result.Shipments))
	for i, s := range result.Shipments {
		shipments[i] = shipmentToProto(s)
	}

	return &pb.ListShipmentsResponse{
		Shipments: shipments,
		PageInfo: &pb.PageInfo{
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
		},
	}, nil
}

// GetShipment returns a single shipment by ID.
func (h *shippingGRPCHandler) GetShipment(ctx context.Context, req *pb.GetShipmentRequest) (*pb.GetShipmentResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.GetShipmentParams{
		ShipmentID: req.Id,
		Includes:   req.Includes,
	}

	shipment, apiErr := h.shipmentSvc.GetShipment(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetShipmentResponse{
		Shipment: shipmentToProto(shipment),
	}, nil
}

// UpdateShipment updates a shipment's mutable fields.
func (h *shippingGRPCHandler) UpdateShipment(ctx context.Context, req *pb.UpdateShipmentRequest) (*pb.UpdateShipmentResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateShipmentParams{
		ShipmentID: req.Id,
		Includes:   req.Includes,
	}

	if req.Note != nil {
		params.Note = req.Note
	}
	if req.Number != nil {
		params.Number = req.Number
	}
	if req.MasterTrackingNumber != nil {
		params.MasterTrackingNumber = req.MasterTrackingNumber
	}
	if req.CarrierId != nil {
		params.CarrierID = req.CarrierId
	}
	params.ServiceLevelID = field.StringClearableFromProto(req.ServiceLevelId)

	shipment, apiErr := h.shipmentSvc.UpdateShipment(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateShipmentResponse{
		Shipment: shipmentToProto(shipment),
	}, nil
}

// Corrects the tracking and routing of a shipment that has already shipped.
func (h *shippingGRPCHandler) AdminUpdateShipmentTracking(ctx context.Context, req *pb.AdminUpdateShipmentTrackingRequest) (*pb.AdminUpdateShipmentTrackingResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.AdminUpdateShipmentTrackingParams{
		ShipmentID:           req.Id,
		MasterTrackingNumber: req.MasterTrackingNumber,
		CarrierID:            req.CarrierId,
		ServiceLevelID:       field.StringClearableFromProto(req.ServiceLevelId),
		Includes:             req.Includes,
	}

	shipment, apiErr := h.shipmentSvc.AdminUpdateShipmentTracking(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.AdminUpdateShipmentTrackingResponse{
		Shipment: shipmentToProto(shipment),
	}, nil
}

// DeleteShipment deletes a shipment by its ID.
func (h *shippingGRPCHandler) DeleteShipment(ctx context.Context, req *pb.DeleteShipmentRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.DeleteShipmentParams{
		ShipmentID: req.Id,
	}

	apiErr := h.shipmentSvc.DeleteShipment(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}

// ShipShipment marks a shipment as shipped.
func (h *shippingGRPCHandler) ShipShipment(ctx context.Context, req *pb.ShipShipmentRequest) (*pb.ShipShipmentResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.ShipShipmentParams{
		ShipmentID:    req.Id,
		EmailCustomer: req.EmailCustomer,
		Includes:      req.Includes,
	}

	shipment, apiErr := h.shipmentSvc.ShipShipment(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.ShipShipmentResponse{
		Shipment: shipmentToProto(shipment),
	}, nil
}

// VoidShipment voids a shipment.
func (h *shippingGRPCHandler) VoidShipment(ctx context.Context, req *pb.VoidShipmentRequest) (*pb.VoidShipmentResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.VoidShipmentParams{
		ShipmentID: req.Id,
	}

	shipment, apiErr := h.shipmentSvc.VoidShipment(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.VoidShipmentResponse{
		Shipment: shipmentToProto(shipment),
	}, nil
}

// EstimateRate estimates a shipping rate for a given carrier and parcels.
func (h *shippingGRPCHandler) EstimateRate(ctx context.Context, req *pb.EstimateRateRequest) (*pb.EstimateRateResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.EstimateRateParams{
		CarrierID:      req.CarrierId,
		ServiceLevelID: req.ServiceLevelId,
		FromAddress:    addressInputToDomain(req.From),
		ToAddress:      addressInputToDomain(req.To),
	}

	if len(req.ProductLineIds) > 0 {
		params.ProductLineIDs = req.ProductLineIds
	}
	if req.CustomerId != nil {
		params.CustomerID = req.CustomerId
	}
	if req.OrderTotal != nil {
		params.OrderTotal = req.OrderTotal
	}

	parcels := make([]domain.Parcel, len(req.Parcels))
	for i, p := range req.Parcels {
		parcels[i] = parcelInfoToDomain(p)
	}
	params.Parcels = parcels

	rate, apiErr := h.shipmentSvc.EstimateRate(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.EstimateRateResponse{
		Rate: rate,
	}, nil
}

// RateShop returns available carrier options with rates for the given parcels.
func (h *shippingGRPCHandler) RateShop(ctx context.Context, req *pb.RateShopRequest) (*pb.RateShopResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.RateShopParams{
		FromAddress: addressInputToDomain(req.From),
		ToAddress:   addressInputToDomain(req.To),
	}

	if len(req.ProductLineIds) > 0 {
		params.ProductLineIDs = req.ProductLineIds
	}
	if req.CustomerId != nil {
		params.CustomerID = req.CustomerId
	}
	if req.OrderTotal != nil {
		params.OrderTotal = req.OrderTotal
	}

	parcels := make([]domain.Parcel, len(req.Parcels))
	for i, p := range req.Parcels {
		parcels[i] = parcelInfoToDomain(p)
	}
	params.Parcels = parcels

	result, apiErr := h.shipmentSvc.RateShop(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	options := make([]*pb.RateShopOptionInfo, len(result.Options))
	for i, o := range result.Options {
		options[i] = rateShopOptionToProto(o)
	}

	resp := &pb.RateShopResponse{
		Options: options,
	}

	if result.ExemptionType != nil {
		resp.ExemptionType = result.ExemptionType
	}
	if result.FlatRate != nil {
		resp.FlatRate = result.FlatRate
	}

	return resp, nil
}

// ListShipmentLines returns a paginated list of lines for a shipment.
func (h *shippingGRPCHandler) ListShipmentLines(ctx context.Context, req *pb.ListShipmentLinesRequest) (*pb.ListShipmentLinesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListShipmentLinesParams{
		ShipmentID: req.ShipmentId,
		Limit:      req.Limit,
		Query:      req.Query,
	}

	if req.Cursor != nil {
		params.Cursor = req.Cursor
	}

	result, apiErr := h.shipmentLineSvc.ListShipmentLines(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	lines := make([]*pb.ShipmentLineInfo, len(result.Lines))
	for i, l := range result.Lines {
		lines[i] = shipmentLineToProto(l)
	}

	return &pb.ListShipmentLinesResponse{
		ShipmentLines: lines,
		PageInfo: &pb.PageInfo{
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
		},
	}, nil
}

// GetShipmentLine returns a single shipment line by its ID.
func (h *shippingGRPCHandler) GetShipmentLine(ctx context.Context, req *pb.GetShipmentLineRequest) (*pb.GetShipmentLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	line, apiErr := h.shipmentLineSvc.GetShipmentLine(ctx, "", req.ShipmentId, req.Id)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetShipmentLineResponse{
		ShipmentLine: shipmentLineToProto(line),
	}, nil
}

// CreateShipmentLine creates a new line on a shipment.
func (h *shippingGRPCHandler) CreateShipmentLine(ctx context.Context, req *pb.CreateShipmentLineRequest) (*pb.CreateShipmentLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.CreateShipmentLineEndpointParams{
		ShipmentID:       req.ShipmentId,
		SalesOrderLineID: req.SalesOrderLineId,
		QuantityValue:    req.QuantityValue,
		QuantityUnitID:   req.QuantityUnitId,
	}

	line, apiErr := h.shipmentLineSvc.CreateShipmentLine(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateShipmentLineResponse{
		ShipmentLine: shipmentLineToProto(line),
	}, nil
}

// UpdateShipmentLine updates a shipment line's quantity.
func (h *shippingGRPCHandler) UpdateShipmentLine(ctx context.Context, req *pb.UpdateShipmentLineRequest) (*pb.UpdateShipmentLineResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateShipmentLineEndpointParams{
		ShipmentID:     req.ShipmentId,
		ShipmentLineID: req.Id,
	}

	if req.QuantityValue != nil {
		params.QuantityValue = req.QuantityValue
	}
	if req.QuantityUnitId != nil {
		params.QuantityUnitID = req.QuantityUnitId
	}

	line, apiErr := h.shipmentLineSvc.UpdateShipmentLine(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateShipmentLineResponse{
		ShipmentLine: shipmentLineToProto(line),
	}, nil
}

// DeleteShipmentLine deletes a shipment line.
func (h *shippingGRPCHandler) DeleteShipmentLine(ctx context.Context, req *pb.DeleteShipmentLineRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.DeleteShipmentLineEndpointParams{
		ShipmentID:     req.ShipmentId,
		ShipmentLineID: req.Id,
	}

	apiErr := h.shipmentLineSvc.DeleteShipmentLine(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &emptypb.Empty{}, nil
}
