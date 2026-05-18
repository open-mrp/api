package shipmentep

import (
	"context"
	"fmt"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func ShipmentPresenter(s *pb.ShipmentInfo) apiresource.ShipmentDetail {
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
			lines[i] = ShipmentLinePresenter(l)
		}
		result.Lines = apiresource.NewList(lines, apiresource.PageInfo{})
	}

	if s.ShippingCases != nil {
		cases := make([]apiresource.ShippingCaseDetail, len(s.ShippingCases))
		for i, c := range s.ShippingCases {
			cases[i] = ShippingCaseDetailPresenter(c)
		}
		result.ShippingCases = apiresource.NewList(cases, apiresource.PageInfo{})
	}

	return result
}

func ShipmentSummaryPresenter(s *pb.ShipmentSummaryInfo) apiresource.ShipmentSummary {
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

func ShipmentListPresenter(ctx context.Context, resp *pb.ListShipmentsResponse) *apiresource.List[apiresource.ShipmentSummary] {
	if resp == nil {
		return apiresource.NewList[apiresource.ShipmentSummary](nil, apiresource.PageInfo{})
	}

	summaries := make([]apiresource.ShipmentSummary, len(resp.Shipments))
	for i, s := range resp.Shipments {
		summaries[i] = ShipmentSummaryPresenter(s)
	}

	return apiresource.NewList(summaries, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}

func ShipmentLinePresenter(l *pb.ShipmentLineInfo) apiresource.ShipmentLine {
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
			Unit: &apiresource.Unit{
				ID:           l.QuantityUnitId,
				Object:       constants.ObjectTypeUnit,
				Name:         l.QuantityUnitName,
				Abbreviation: l.QuantityUnitAbbreviation,
			},
		},
		CreatedAt: grpcutil.TimestampToTime(l.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(l.UpdatedAt),
	}

	if l.SalesOrderLineId != "" {
		result.SalesOrderLine = &apiresource.SalesOrderLineDetail{
			ID:                 l.SalesOrderLineId,
			Object:             constants.ObjectTypeSalesOrderLine,
			ProductSKU:         l.OrderLineSku,
			ProductDescription: l.OrderLineDescription,
		}
	}

	return result
}

func ShipmentLineListPresenter(ctx context.Context, resp *pb.ListShipmentLinesResponse) *apiresource.List[apiresource.ShipmentLine] {
	if resp == nil {
		return apiresource.NewList[apiresource.ShipmentLine](nil, apiresource.PageInfo{})
	}

	lines := make([]apiresource.ShipmentLine, len(resp.ShipmentLines))
	for i, l := range resp.ShipmentLines {
		lines[i] = ShipmentLinePresenter(l)
	}

	return apiresource.NewList(lines, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}

func ShippingCaseDetailPresenter(c *pb.ShippingCaseDetailInfo) apiresource.ShippingCaseDetail {
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
			Unit: &apiresource.Unit{
				ID:           c.FreightAmountUnitId,
				Object:       constants.ObjectTypeUnit,
				Name:         c.FreightAmountUnitName,
				Abbreviation: c.FreightAmountUnitAbbreviation,
			},
		}
	}

	if c.FreightWeightId != "" {
		result.FreightWeight = &apiresource.Quantity{
			ID:           c.FreightWeightId,
			Object:       constants.ObjectTypeQuantity,
			Value:        c.FreightWeightValue,
			DisplayValue: fmt.Sprintf("%s %s", c.FreightWeightValue, c.FreightWeightUnitAbbreviation),
			Unit: &apiresource.Unit{
				ID:           c.FreightWeightUnitId,
				Object:       constants.ObjectTypeUnit,
				Name:         c.FreightWeightUnitName,
				Abbreviation: c.FreightWeightUnitAbbreviation,
			},
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

func EstimateRatePresenter(resp *pb.EstimateRateResponse) *apiresource.EstimateRateResult {
	if resp == nil {
		return nil
	}
	return &apiresource.EstimateRateResult{
		Object: constants.ObjectTypeEstimateRateResult,
		Rate:   resp.Rate,
	}
}

func RateShopPresenter(resp *pb.RateShopResponse) *apiresource.RateShopResult {
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
