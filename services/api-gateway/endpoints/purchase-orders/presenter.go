package purchaseorderep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func PurchaseOrderSummaryPresenter(info *pb.PurchaseOrderSummaryInfo) apiresource.PurchaseOrderSummary {
	s := apiresource.PurchaseOrderSummary{
		ID:     info.Id,
		Object: constants.ObjectTypePurchaseOrder,
		Number: info.Number,
		Supplier: &apiresource.Supplier{
			ID:     info.SupplierId,
			Object: constants.ObjectTypeSupplier,
			Name:   info.SupplierName,
			Number: info.SupplierNumber,
		},
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

func PurchaseOrderDetailPresenter(info *pb.PurchaseOrderInfo) apiresource.PurchaseOrderDetail {
	d := apiresource.PurchaseOrderDetail{
		ID:                    info.Id,
		Object:                constants.ObjectTypePurchaseOrder,
		Number:                info.Number,
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

	// Supplier
	d.Supplier = &apiresource.Supplier{
		ID:     info.SupplierId,
		Object: constants.ObjectTypeSupplier,
		Name:   info.SupplierName,
		Number: info.SupplierNumber,
	}

	// Bill-to address
	if info.BillingAddressId != "" {
		d.BillToAddress = buildAddressFromProto(
			info.BillingAddressId, info.BillToName, info.BillToStreetLine_1, info.BillToStreetLine_2,
			info.BillToLocality, info.BillToState, info.BillToPostalCode, info.BillToCountry,
			info.BillToPhone, info.BillToEmail,
		)
	}

	// Ship-to address
	if info.ShippingAddressId != "" {
		d.ShipToAddress = buildAddressFromProto(
			info.ShippingAddressId, info.ShipToName, info.ShipToStreetLine_1, info.ShipToStreetLine_2,
			info.ShipToLocality, info.ShipToState, info.ShipToPostalCode, info.ShipToCountry,
			info.ShipToPhone, info.ShipToEmail,
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
	}

	// Receiving order
	if info.ReceivingOrderId != nil {
		d.ReceivingOrder = &apiresource.ReceivingOrder{
			ID:     *info.ReceivingOrderId,
			Object: constants.ObjectTypeReceivingOrder,
		}
	}

	// Contacts
	if len(info.Contacts) > 0 {
		contactItems := make([]apiresource.EmailContact, len(info.Contacts))
		for i, c := range info.Contacts {
			contactItems[i] = apiresource.EmailContact{
				ID:     c.Id,
				Object: constants.ObjectTypeEmailContact,
				AccountUser: &apiresource.AccountUser{
					ID:     c.AccountUserId,
					Object: constants.ObjectTypeAccountUser,
				},
			}
		}
		d.Contacts = apiresource.NewList(contactItems, apiresource.PageInfo{})
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
	if info.PromisedAt != nil {
		t := grpcutil.TimestampToTime(info.PromisedAt)
		d.ScheduledAt = &t
	}

	// Lines
	if len(info.Lines) > 0 {
		lines := make([]apiresource.PurchaseOrderLineDetail, len(info.Lines))
		for i, l := range info.Lines {
			lines[i] = PurchaseOrderLineDetailPresenter(l)
		}
		d.Lines = apiresource.NewList(lines, apiresource.PageInfo{})
	}

	return d
}

func PurchaseOrderLineDetailPresenter(info *pb.PurchaseOrderLineInfo) apiresource.PurchaseOrderLineDetail {
	l := apiresource.PurchaseOrderLineDetail{
		ID:                 info.Id,
		Object:             constants.ObjectTypePurchaseOrderLine,
		LineItemNumber:     info.LineItemNumber,
		ProductSKU:         info.ProductSku,
		ProductDescription: info.ProductDescription,
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

	// Quantity received
	if info.QuantityReceivedValue != nil {
		l.QuantityReceived = &apiresource.Quantity{
			Object: constants.ObjectTypeQuantity,
			Value:  *info.QuantityReceivedValue,
			Unit: &apiresource.Unit{
				ID:           info.QuantityUnitId,
				Object:       constants.ObjectTypeUnit,
				Name:         info.QuantityUnitName,
				Abbreviation: info.QuantityUnitAbbreviation,
			},
		}
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

	return l
}

func PurchaseOrderStatusListPresenter(resp *pb.ListSalesOrderStatusesResponse) *apiresource.List[apiresource.SalesOrderStatus] {
	if resp == nil {
		return apiresource.NewList[apiresource.SalesOrderStatus](nil, apiresource.PageInfo{})
	}

	statuses := make([]apiresource.SalesOrderStatus, len(resp.SalesOrderStatuses))
	for i, s := range resp.SalesOrderStatuses {
		statuses[i] = apiresource.SalesOrderStatus{
			ID:        s.Id,
			Object:    constants.ObjectTypeSalesOrderStatus,
			Code:      constants.SalesOrderStatusCode(s.Code),
			Name:      s.Name,
			CreatedAt: grpcutil.TimestampToTime(s.CreatedAt),
			UpdatedAt: grpcutil.TimestampToTime(s.UpdatedAt),
		}
	}

	return apiresource.NewList(statuses, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func PurchaseOrderListPresenter(resp *pb.ListPurchaseOrdersResponse) *apiresource.List[apiresource.PurchaseOrderSummary] {
	orders := make([]apiresource.PurchaseOrderSummary, len(resp.PurchaseOrders))
	for i, o := range resp.PurchaseOrders {
		orders[i] = PurchaseOrderSummaryPresenter(o)
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
) *apiresource.Address {
	addr := &apiresource.Address{
		ID:     id,
		Object: constants.ObjectTypeAddress,
		Phone:  phone,
		Email:  email,
	}

	if name != nil {
		addr.Name = *name
	}

	countryStr := ""
	if country != nil {
		countryStr = *country
	}

	addr.Geolocation = &apiresource.Geolocation{
		Object:      constants.ObjectTypeGeolocation,
		StreetLine1: line1,
		StreetLine2: line2,
		Locality:    locality,
		State:       state,
		PostalCode:  postalCode,
		Country:     countryStr,
	}

	return addr
}
