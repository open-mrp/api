package salesorderep

import (
	"time"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func SalesOrderSummaryPresenter(info *pb.SalesOrderSummaryInfo) apiresource.SalesOrderSummary {
	customer := &apiresource.Customer{
		ID:               info.CustomerId,
		Object:           constants.ObjectTypeCustomer,
		Name:             info.CustomerName,
		Number:           info.CustomerNumber,
		EDIStatus:        constants.EDIStatusDisabled,
		RelationshipType: constants.CustomerRelationshipTypeStandalone,
	}
	if info.CustomerStatusCode != nil {
		customer.Status = constants.AccountStatusCode(*info.CustomerStatusCode)
	}
	if info.CustomerCommissionPolicy != nil {
		customer.CommissionPolicy = constants.CommissionPolicy(*info.CustomerCommissionPolicy)
	}

	s := apiresource.SalesOrderSummary{
		ID:       info.Id,
		Object:   constants.ObjectTypeSalesOrder,
		Number:   info.Number,
		Customer: customer,
		Status: &apiresource.SalesOrderStatusDetail{
			Code:   info.StatusCode,
			Object: constants.ObjectTypeSalesOrderStatus,
			Name:   info.StatusName,
		},
		Type: &apiresource.SalesOrderType{
			Code:   info.TypeCode,
			Object: constants.ObjectTypeSalesOrderType,
			Name:   info.TypeName,
		},
		Priority: &apiresource.Priority{
			Code:   constants.PriorityCode(info.PriorityCode),
			Object: constants.ObjectTypePriority,
			Name:   info.PriorityName,
		},
		LineCount:            info.LineCount,
		IsAcknowledgmentSent: info.IsAcknowledgmentSent,
		CustomerPO:           info.CustomerPoNumber,
		CreatedAt:            grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.PriorityId != nil {
		s.Priority.ID = *info.PriorityId
	}

	if info.IssuedAt != nil {
		t := grpcutil.TimestampToTime(info.IssuedAt)
		s.IssuedAt = &t
	}
	if info.CompletedAt != nil {
		t := grpcutil.TimestampToTime(info.CompletedAt)
		s.CompletedAt = &t
	}

	return s
}

func SalesOrderDetailPresenter(info *pb.SalesOrderInfo) apiresource.SalesOrderDetail {
	d := apiresource.SalesOrderDetail{
		ID:                    info.Id,
		Object:                constants.ObjectTypeSalesOrder,
		Number:                info.Number,
		CustomerPO:            info.CustomerPoNumber,
		Note:                  info.Note,
		IsAcknowledgmentSent:  info.IsAcknowledgmentSent,
		CarrierBillingType:    info.CarrierBillingType,
		CarrierBillingAccount: info.CarrierBillingAccount,
		Status: &apiresource.SalesOrderStatusDetail{
			Code:   info.StatusCode,
			Object: constants.ObjectTypeSalesOrderStatus,
			Name:   info.StatusName,
		},
		Type: &apiresource.SalesOrderType{
			Code:   info.TypeCode,
			Object: constants.ObjectTypeSalesOrderType,
			Name:   info.TypeName,
		},
		Priority: &apiresource.Priority{
			Code:   constants.PriorityCode(info.PriorityCode),
			Object: constants.ObjectTypePriority,
			Name:   info.PriorityName,
		},
		CreatedAt: grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}

	if info.PriorityId != nil {
		d.Priority.ID = *info.PriorityId
	}

	// Customer
	d.Customer = &apiresource.Customer{
		ID:               info.CustomerId,
		Object:           constants.ObjectTypeCustomer,
		Name:             info.CustomerName,
		Number:           info.CustomerNumber,
		EDIStatus:        constants.EDIStatusDisabled,
		RelationshipType: constants.CustomerRelationshipTypeStandalone,
	}
	if info.CustomerStatusCode != nil {
		d.Customer.Status = constants.AccountStatusCode(*info.CustomerStatusCode)
	}
	if info.CustomerCommissionPolicy != nil {
		d.Customer.CommissionPolicy = constants.CommissionPolicy(*info.CustomerCommissionPolicy)
	}
	if info.CustomerCreatedAt != nil {
		d.Customer.CreatedAt = info.CustomerCreatedAt.AsTime()
	}
	if info.CustomerUpdatedAt != nil {
		d.Customer.UpdatedAt = info.CustomerUpdatedAt.AsTime()
	}

	// Bill-to address
	if info.BillingAddressId != "" {
		d.BillToAddress = buildAddressFromProto(
			info.BillingAddressId, info.BillToName, info.BillToStreetLine_1, info.BillToStreetLine_2,
			info.BillToLocality, info.BillToState, info.BillToPostalCode, info.BillToCountry,
			info.BillToPhone, info.BillToEmail,
			info.BillToIsDropShip, info.BillToGeolocationId,
			grpcutil.TimestampToTime(info.BillToCreatedAt), grpcutil.TimestampToTime(info.BillToUpdatedAt),
		)
	}

	// Ship-to address
	if info.ShippingAddressId != "" {
		d.ShipToAddress = buildAddressFromProto(
			info.ShippingAddressId, info.ShipToName, info.ShipToStreetLine_1, info.ShipToStreetLine_2,
			info.ShipToLocality, info.ShipToState, info.ShipToPostalCode, info.ShipToCountry,
			info.ShipToPhone, info.ShipToEmail,
			info.ShipToIsDropShip, info.ShipToGeolocationId,
			grpcutil.TimestampToTime(info.ShipToCreatedAt), grpcutil.TimestampToTime(info.ShipToUpdatedAt),
		)
	}

	// Carrier
	if info.CarrierId != nil {
		d.Carrier = &apiresource.Carrier{
			ID:     *info.CarrierId,
			Object: constants.ObjectTypeCarrier,
		}
		if info.CarrierName != nil {
			d.Carrier.Name = *info.CarrierName
		}
		if info.CarrierIsPortalEnabled != nil && *info.CarrierIsPortalEnabled {
			d.Carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityVisible
		} else {
			d.Carrier.CustomerPortalVisibility = constants.CustomerPortalVisibilityHidden
		}
		if info.CarrierCreatedAt != nil {
			d.Carrier.CreatedAt = info.CarrierCreatedAt.AsTime()
		}
		if info.CarrierUpdatedAt != nil {
			d.Carrier.UpdatedAt = info.CarrierUpdatedAt.AsTime()
		}
	}

	// Service level
	if info.ServiceLevelId != nil {
		d.ServiceLevel = &apiresource.ServiceLevel{
			ID:     *info.ServiceLevelId,
			Object: constants.ObjectTypeServiceLevel,
		}
		if info.ServiceLevelName != nil {
			d.ServiceLevel.Name = *info.ServiceLevelName
		}
		if info.ServiceLevelIsPortalEnabled != nil && *info.ServiceLevelIsPortalEnabled {
			d.ServiceLevel.CustomerPortalVisibility = constants.CustomerPortalVisibilityVisible
		} else {
			d.ServiceLevel.CustomerPortalVisibility = constants.CustomerPortalVisibilityHidden
		}
		if info.ServiceLevelToken != nil {
			d.ServiceLevel.ServiceLevelToken = constants.ServiceLevelCode(*info.ServiceLevelToken)
		}
		if info.ServiceLevelCreatedAt != nil {
			d.ServiceLevel.CreatedAt = info.ServiceLevelCreatedAt.AsTime()
		}
		if info.ServiceLevelUpdatedAt != nil {
			d.ServiceLevel.UpdatedAt = info.ServiceLevelUpdatedAt.AsTime()
		}
	}

	// Sales rep (as Actor sub-resource)
	if info.SalesRepId != nil {
		d.SalesRep = apiresource.NewActor(
			*info.SalesRepId,
			constants.ActorTypeUser,
			info.SalesRepName,
			nil,
		)
	}

	// Payment term
	if info.PaymentTermId != nil {
		d.PaymentTerm = &apiresource.PaymentTerm{
			ID:     *info.PaymentTermId,
			Object: constants.ObjectTypePaymentTerm,
		}
		if info.PaymentTermName != nil {
			d.PaymentTerm.Name = *info.PaymentTermName
		}
		if info.PaymentTermIsActive != nil && *info.PaymentTermIsActive {
			d.PaymentTerm.Status = constants.PaymentTermStatusActive
		} else {
			d.PaymentTerm.Status = constants.PaymentTermStatusInactive
		}
		if info.PaymentTermCreatedAt != nil {
			d.PaymentTerm.CreatedAt = info.PaymentTermCreatedAt.AsTime()
		}
		if info.PaymentTermUpdatedAt != nil {
			d.PaymentTerm.UpdatedAt = info.PaymentTermUpdatedAt.AsTime()
		}
	}

	// Shipping term
	if info.ShippingTermId != nil {
		d.ShippingTerm = &apiresource.ShippingTerm{
			ID:     *info.ShippingTermId,
			Object: constants.ObjectTypeShippingTerm,
		}
		if info.ShippingTermName != nil {
			d.ShippingTerm.Name = *info.ShippingTermName
		}
		if info.ShippingTermType != nil {
			d.ShippingTerm.Type = constants.ShippingTermType(*info.ShippingTermType)
		}
		if info.ShippingTermCreatedAt != nil {
			d.ShippingTerm.CreatedAt = info.ShippingTermCreatedAt.AsTime()
		}
		if info.ShippingTermUpdatedAt != nil {
			d.ShippingTerm.UpdatedAt = info.ShippingTermUpdatedAt.AsTime()
		}
	}

	// Order discount
	if info.OrderDiscountId != nil {
		d.OrderDiscount = &apiresource.OrderDiscount{
			ID:     *info.OrderDiscountId,
			Object: constants.ObjectTypeOrderDiscount,
		}
		if info.OrderDiscountName != nil {
			d.OrderDiscount.Name = *info.OrderDiscountName
		}
		if info.OrderDiscountCode != nil {
			d.OrderDiscount.Code = *info.OrderDiscountCode
		}
		if info.OrderDiscountPercentage != nil {
			d.OrderDiscount.Percentage = *info.OrderDiscountPercentage
		}
		if info.OrderDiscountAmount != nil {
			d.OrderDiscount.Amount = *info.OrderDiscountAmount
		}
		if info.OrderDiscountDiscountType != nil {
			d.OrderDiscount.DiscountType = constants.OrderDiscountType(*info.OrderDiscountDiscountType)
		}
		if info.OrderDiscountOrderCount != nil {
			d.OrderDiscount.OrderCount = *info.OrderDiscountOrderCount
		}
		if info.OrderDiscountCreatedAt != nil {
			d.OrderDiscount.CreatedAt = info.OrderDiscountCreatedAt.AsTime()
		}
		if info.OrderDiscountUpdatedAt != nil {
			d.OrderDiscount.UpdatedAt = info.OrderDiscountUpdatedAt.AsTime()
		}
	}

	// Production run
	if info.ProductionRunId != nil {
		d.ProductionRun = &apiresource.ProductionRun{
			ID:     *info.ProductionRunId,
			Object: constants.ObjectTypeProductionRun,
		}
	}

	// Pick
	if info.PickId != nil {
		d.Pick = &apiresource.Pick{
			ID:     *info.PickId,
			Object: "pick",
		}
	}

	// Timestamps
	if info.IssuedAt != nil {
		t := grpcutil.TimestampToTime(info.IssuedAt)
		d.IssuedAt = &t
	}
	if info.CompletedAt != nil {
		t := grpcutil.TimestampToTime(info.CompletedAt)
		d.CompletedAt = &t
	}
	if info.FirstShipAt != nil {
		t := grpcutil.TimestampToTime(info.FirstShipAt)
		d.FirstShipAt = &t
	}
	if info.ExpiredAt != nil {
		t := grpcutil.TimestampToTime(info.ExpiredAt)
		d.ExpiredAt = &t
	}
	if info.PromisedAt != nil {
		t := grpcutil.TimestampToTime(info.PromisedAt)
		d.PromisedAt = &t
	}

	// Lines
	if len(info.Lines) > 0 {
		lines := make([]apiresource.SalesOrderLineDetail, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = SalesOrderLineDetailPresenter(l)
		}
		d.Lines = apiresource.NewList(lines, apiresource.PageInfo{})
	}

	return d
}

func SalesOrderLineDetailPresenter(info *pb.SalesOrderLineInfo) apiresource.SalesOrderLineDetail {
	l := apiresource.SalesOrderLineDetail{
		ID:                 info.Id,
		Object:             constants.ObjectTypeSalesOrderLine,
		LineItemNumber:     info.LineItemNumber,
		ProductSKU:         info.ProductSku,
		ProductDescription: info.ProductDescription,
		EdiLineItemID:      info.EdiLineItemId,
		CreatedAt:          grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:          grpcutil.TimestampToTime(info.UpdatedAt),
	}

	// Item
	if info.ItemId != nil {
		item := &apiresource.Item{
			ID:     *info.ItemId,
			Object: constants.ObjectTypeItem,
		}
		if info.ItemSku != nil {
			item.SKU = *info.ItemSku
		}
		l.Item = item
	}

	// Quantity ordered
	l.QuantityOrdered = &apiresource.Quantity{
		ID:     info.QuantityId,
		Object: constants.ObjectTypeQuantity,
		Value:  info.QuantityValue,
		Unit: &apiresource.Unit{
			ID:           info.QuantityUnitId,
			Object:       constants.ObjectTypeUnit,
			Name:         info.QuantityUnitName,
			Abbreviation: info.QuantityUnitAbbreviation,
		},
	}

	// Unit price
	l.UnitPrice = &apiresource.Rate{
		ID:     info.UnitPriceId,
		Object: constants.ObjectTypeRate,
		Value:  info.UnitPriceValue,
		NumeratorUnit: &apiresource.Unit{
			ID:           info.UnitPriceNumeratorUnitId,
			Object:       constants.ObjectTypeUnit,
			Abbreviation: info.UnitPriceNumeratorUnitAbbreviation,
		},
		DenominatorUnit: &apiresource.Unit{
			ID:           info.UnitPriceDenominatorUnitId,
			Object:       constants.ObjectTypeUnit,
			Abbreviation: info.UnitPriceDenominatorUnitAbbreviation,
		},
		DisplayValue: apiresource.FormatRateDisplayValue(info.UnitPriceValue, info.UnitPriceNumeratorUnitAbbreviation, "", info.UnitPriceDenominatorUnitAbbreviation),
	}

	// Unit cost
	if info.UnitCostId != nil {
		l.UnitCost = &apiresource.Rate{
			ID:     *info.UnitCostId,
			Object: constants.ObjectTypeRate,
		}
		var unitCostValue, unitCostNumeratorAbbr, unitCostDenominatorAbbr string
		if info.UnitCostValue != nil {
			l.UnitCost.Value = *info.UnitCostValue
			unitCostValue = *info.UnitCostValue
		}
		if info.UnitCostNumeratorUnitId != nil {
			l.UnitCost.NumeratorUnit = &apiresource.Unit{
				ID:     *info.UnitCostNumeratorUnitId,
				Object: constants.ObjectTypeUnit,
			}
			if info.UnitCostNumeratorUnitAbbreviation != nil {
				l.UnitCost.NumeratorUnit.Abbreviation = *info.UnitCostNumeratorUnitAbbreviation
				unitCostNumeratorAbbr = *info.UnitCostNumeratorUnitAbbreviation
			}
		}
		if info.UnitCostDenominatorUnitId != nil {
			l.UnitCost.DenominatorUnit = &apiresource.Unit{
				ID:     *info.UnitCostDenominatorUnitId,
				Object: constants.ObjectTypeUnit,
			}
			if info.UnitCostDenominatorUnitAbbreviation != nil {
				l.UnitCost.DenominatorUnit.Abbreviation = *info.UnitCostDenominatorUnitAbbreviation
				unitCostDenominatorAbbr = *info.UnitCostDenominatorUnitAbbreviation
			}
		}
		l.UnitCost.DisplayValue = apiresource.FormatRateDisplayValue(unitCostValue, unitCostNumeratorAbbr, "", unitCostDenominatorAbbr)
	}

	// Quantity picked
	if info.QuantityPickedValue != nil {
		l.QuantityPicked = &apiresource.Quantity{
			Object: constants.ObjectTypeQuantity,
			Value:  *info.QuantityPickedValue,
			Unit:   l.QuantityOrdered.Unit,
		}
	}

	// Quantity packed
	if info.QuantityPackedValue != nil {
		l.QuantityPacked = &apiresource.Quantity{
			Object: constants.ObjectTypeQuantity,
			Value:  *info.QuantityPackedValue,
			Unit:   l.QuantityOrdered.Unit,
		}
	}

	// Quantity invoiced
	if info.QuantityInvoicedValue != nil {
		l.QuantityInvoiced = &apiresource.Quantity{
			Object: constants.ObjectTypeQuantity,
			Value:  *info.QuantityInvoicedValue,
			Unit:   l.QuantityOrdered.Unit,
		}
	}

	// Completed at
	if info.CompletedAt != nil {
		t := grpcutil.TimestampToTime(info.CompletedAt)
		l.CompletedAt = &t
	}

	return l
}

func SalesOrderListPresenter(resp *pb.ListSalesOrdersResponse) *apiresource.List[apiresource.SalesOrderSummary] {
	orders := make([]apiresource.SalesOrderSummary, len(resp.SalesOrders))
	for i, o := range resp.SalesOrders {
		orders[i] = SalesOrderSummaryPresenter(o)
	}

	return apiresource.NewList(orders, apiresource.PageInfo{
		NextCursor:  resp.PageInfo.NextCursor,
		PrevCursor:  resp.PageInfo.PrevCursor,
		HasNextPage: resp.PageInfo.HasNextPage,
		HasPrevPage: resp.PageInfo.HasPrevPage,
	})
}

func buildAddressFromProto(
	id string, name, line1, line2, locality, state, postalCode, country, phone, email *string,
	isDropShip *bool, geolocationID *string,
	createdAt time.Time, updatedAt time.Time,
) *apiresource.Address {
	addr := &apiresource.Address{
		ID:        id,
		Object:    constants.ObjectTypeAddress,
		Phone:     phone,
		Email:     email,
		Type:      constants.AddressTypeStandard,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	if name != nil {
		addr.Name = *name
	}

	if isDropShip != nil && *isDropShip {
		addr.Type = constants.AddressTypeDropShip
	}

	countryStr := ""
	if country != nil {
		countryStr = *country
	}

	geo := &apiresource.Geolocation{
		Object:      constants.ObjectTypeGeolocation,
		StreetLine1: line1,
		StreetLine2: line2,
		Locality:    locality,
		State:       state,
		PostalCode:  postalCode,
		Country:     countryStr,
	}
	if geolocationID != nil {
		geo.ID = *geolocationID
	}
	addr.Geolocation = geo

	return addr
}
