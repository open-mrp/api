package invoiceep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func InvoiceSummaryPresenter(d *pb.InvoiceSummaryInfo) apiresource.InvoiceSummary {
	if d == nil {
		return apiresource.InvoiceSummary{}
	}

	customer := &apiresource.Customer{
		ID:               d.CustomerId,
		Object:           constants.ObjectTypeCustomer,
		Name:             d.CustomerName,
		Number:           d.CustomerNumber,
		EDIStatus:        constants.EDIStatusDisabled,
		RelationshipType: constants.CustomerRelationshipTypeStandalone,
	}
	if d.CustomerStatusCode != nil {
		customer.Status = constants.AccountStatusCode(*d.CustomerStatusCode)
	}
	if d.CustomerCommissionPolicy != nil {
		customer.CommissionPolicy = constants.CommissionPolicy(*d.CustomerCommissionPolicy)
	}

	summary := apiresource.InvoiceSummary{
		ID:       d.Id,
		Object:   constants.ObjectTypeInvoiceSummary,
		Number:   d.Number,
		Note:     d.Note,
		Customer: customer,
		Order: &apiresource.SalesOrderDetail{
			ID:     d.OrderId,
			Object: constants.ObjectTypeSalesOrder,
			Number: d.OrderNumber,
		},
		LineCount:            d.LineCount,
		BillingAddress:       invoiceBillingAddressPresenter(d.BillingAddressId, d.BillingAddressName, d.BillingAddressLine1, d.BillingAddressLine2, d.BillingAddressCity, d.BillingAddressState, d.BillingAddressZip, d.BillingAddressCountry),
		PriorityCode:         constants.PriorityCode(d.PriorityCode),
		IsPaidInFull:         d.IsPaidInFull,
		IsEdiSent:            d.IsEdiSent,
		HasBeenSent:          d.HasBeenSent,
		TotalInvoiced:        d.TotalInvoiced,
		AcceptsInvoiceEmails: d.AcceptsInvoiceEmails,
		CustomerIsEdiEnabled: d.CustomerIsEdiEnabled,
		CreatedAt:            grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(d.UpdatedAt),
	}

	if d.ShipmentId != nil {
		summary.Shipment = &apiresource.ShipmentDetail{
			ID:     *d.ShipmentId,
			Object: constants.ObjectTypeShipment,
		}
	}

	if d.PaymentTermId != nil {
		pt := &apiresource.PaymentTerm{
			ID:     *d.PaymentTermId,
			Object: constants.ObjectTypePaymentTerm,
		}
		if d.PaymentTermName != nil {
			pt.Name = *d.PaymentTermName
		}
		if d.PaymentTermIsActive != nil && *d.PaymentTermIsActive {
			pt.Status = constants.PaymentTermStatusActive
		} else {
			pt.Status = constants.PaymentTermStatusInactive
		}
		summary.PaymentTerm = pt
	}

	return summary
}

func InvoicePresenter(d *pb.InvoiceInfo) apiresource.Invoice {
	if d == nil {
		return apiresource.Invoice{}
	}

	lines := make([]apiresource.InvoiceLine, len(d.Lines))
	for i, l := range d.Lines {
		lines[i] = InvoiceLinePresenter(l)
	}

	allocations := make([]apiresource.InvoiceAllocation, len(d.Allocations))
	for i, a := range d.Allocations {
		allocations[i] = InvoiceAllocationPresenter(a)
	}

	inv := apiresource.Invoice{
		ID:     d.Id,
		Object: constants.ObjectTypeInvoice,
		Number: d.Number,
		Note:   d.Note,
		Order: &apiresource.SalesOrderDetail{
			ID:     d.OrderId,
			Object: constants.ObjectTypeSalesOrder,
			Number: d.OrderNumber,
		},
		BillingAddress:       invoiceBillingAddressPresenter(d.BillingAddressId, d.BillingAddressName, d.BillingAddressLine1, d.BillingAddressLine2, d.BillingAddressCity, d.BillingAddressState, d.BillingAddressZip, d.BillingAddressCountry),
		IsPaidInFull:         d.IsPaidInFull,
		IsOverPaid:           d.IsOverPaid,
		IsEdiSent:            d.IsEdiSent,
		HasBeenSent:          d.HasBeenSent,
		AcceptsInvoiceEmails: d.AcceptsInvoiceEmails,
		Lines:                apiresource.NewList(lines, apiresource.PageInfo{}),
		Allocations:          apiresource.NewList(allocations, apiresource.PageInfo{}),
		CreatedAt:            grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:            grpcutil.TimestampToTime(d.UpdatedAt),
	}

	if d.ShipmentId != nil {
		shipment := &apiresource.ShipmentDetail{
			ID:     *d.ShipmentId,
			Object: constants.ObjectTypeShipment,
		}
		if d.ShipmentNumber != nil {
			shipment.Number = *d.ShipmentNumber
		}
		inv.Shipment = shipment
	}

	return inv
}

func InvoiceLinePresenter(l *pb.InvoiceLineInfo) apiresource.InvoiceLine {
	if l == nil {
		return apiresource.InvoiceLine{}
	}

	line := apiresource.InvoiceLine{
		ID:     l.Id,
		Object: constants.ObjectTypeInvoiceLine,
		Quantity: &apiresource.Quantity{
			ID:           l.QuantityId,
			Object:       constants.ObjectTypeQuantity,
			Value:        l.QuantityValue,
			DisplayValue: apiresource.FormatDisplayValue(l.QuantityValue, l.QuantityUnitAbbreviation, ""),
			Unit: &apiresource.Unit{
				ID:     l.QuantityUnitId,
				Object: constants.ObjectTypeUnit,
			},
		},
		UnitPrice: &apiresource.Rate{
			ID:     l.UnitPriceId,
			Object: constants.ObjectTypeRate,
			Value:  l.UnitPriceValue,
			NumeratorUnit: &apiresource.Unit{
				ID:     l.UnitPriceNumeratorUnitId,
				Object: constants.ObjectTypeUnit,
			},
			DenominatorUnit: &apiresource.Unit{
				ID:     l.UnitPriceDenominatorUnitId,
				Object: constants.ObjectTypeUnit,
			},
			DisplayValue: "",
		},
		OrderLine: &apiresource.SalesOrderLineDetail{
			ID:     l.OrderLineId,
			Object: constants.ObjectTypeSalesOrderLine,
		},
		CreatedAt: grpcutil.TimestampToTime(l.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(l.UpdatedAt),
	}

	if l.OrderLineItemId != nil {
		item := &apiresource.Item{
			ID:     *l.OrderLineItemId,
			Object: constants.ObjectTypeItem,
		}
		if l.OrderLineItemSku != nil {
			item.SKU = *l.OrderLineItemSku
		}
		if line.OrderLine != nil {
			line.OrderLine.Item = item
		}
	}

	return line
}

func InvoiceAllocationPresenter(a *pb.InvoiceAllocationInfo) apiresource.InvoiceAllocation {
	if a == nil {
		return apiresource.InvoiceAllocation{}
	}

	return apiresource.InvoiceAllocation{
		ID:     a.Id,
		Object: constants.ObjectTypeInvoiceAllocation,
		Transaction: &apiresource.TransactionDetail{
			ID:     a.TransactionId,
			Object: constants.ObjectTypeTransaction,
		},
		Amount: &apiresource.Quantity{
			ID:           a.AmountId,
			Object:       constants.ObjectTypeQuantity,
			Value:        a.AmountValue,
			DisplayValue: apiresource.FormatDisplayValue(a.AmountValue, a.AmountUnitAbbreviation, string(constants.UnitTypeCurrency)),
			Unit: &apiresource.Unit{
				ID:     a.AmountUnitId,
				Object: constants.ObjectTypeUnit,
			},
		},
		Note:      a.Note,
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}
}

func InvoiceForPaymentPresenter(d *pb.InvoiceForPaymentInfo) apiresource.InvoiceForPayment {
	if d == nil {
		return apiresource.InvoiceForPayment{}
	}

	allocations := make([]apiresource.InvoiceAllocation, len(d.Allocations))
	for i, a := range d.Allocations {
		allocations[i] = InvoiceAllocationPresenter(a)
	}

	inv := apiresource.InvoiceForPayment{
		ID:         d.Id,
		Object:     constants.ObjectTypeInvoiceForPayment,
		Number:     d.Number,
		CustomerPO: d.CustomerPo,
		Customer: &apiresource.Customer{
			ID:               d.CustomerId,
			Object:           constants.ObjectTypeCustomer,
			Name:             d.CustomerName,
			Number:           d.CustomerNumber,
			EDIStatus:        constants.EDIStatusDisabled,
			RelationshipType: constants.CustomerRelationshipTypeStandalone,
		},
		IsParentAccount: d.IsParentAccount,
		IsPrepaid:       d.IsPrepaid,
		InvoiceTotal:    d.InvoiceTotal,
		IsPaidInFull:    d.IsPaidInFull,
		Allocations:     apiresource.NewList(allocations, apiresource.PageInfo{}),
		CreatedAt:       grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:       grpcutil.TimestampToTime(d.UpdatedAt),
	}

	if d.ParentAccountId != nil {
		inv.ParentAccount = &apiresource.Account{
			ID:     *d.ParentAccountId,
			Object: constants.ObjectTypeAccount,
		}
	}

	if d.BillingAddressId != nil {
		addr := &apiresource.Address{
			ID:     *d.BillingAddressId,
			Object: constants.ObjectTypeAddress,
			Type:   constants.AddressTypeStandard,
		}
		if d.BillingAddressName != nil {
			addr.Name = *d.BillingAddressName
		}
		inv.BillingAddress = addr
	}

	return inv
}

func InvoiceListPresenter(resp *pb.ListInvoicesResponse) *apiresource.List[apiresource.InvoiceSummary] {
	if resp == nil {
		return apiresource.NewList[apiresource.InvoiceSummary](nil, apiresource.PageInfo{})
	}

	invoices := make([]apiresource.InvoiceSummary, len(resp.Invoices))
	for i, d := range resp.Invoices {
		invoices[i] = InvoiceSummaryPresenter(d)
	}

	return apiresource.NewList(invoices, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func CustomerInvoiceListPresenter(resp *pb.ListCustomerInvoicesResponse) *apiresource.List[apiresource.InvoiceForPayment] {
	if resp == nil {
		return apiresource.NewList[apiresource.InvoiceForPayment](nil, apiresource.PageInfo{})
	}

	invoices := make([]apiresource.InvoiceForPayment, len(resp.Invoices))
	for i, d := range resp.Invoices {
		invoices[i] = InvoiceForPaymentPresenter(d)
	}

	return apiresource.NewList(invoices, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func invoiceBillingAddressPresenter(id string, name, line1, line2, city, state, zip *string, country string) *apiresource.Address {
	addr := &apiresource.Address{
		ID:     id,
		Object: constants.ObjectTypeAddress,
		Type:   constants.AddressTypeStandard,
		Geolocation: &apiresource.Geolocation{
			StreetLine1: line1,
			StreetLine2: line2,
			Locality:    city,
			State:       state,
			PostalCode:  zip,
			Country:     country,
		},
	}
	if name != nil {
		addr.Name = *name
	}
	return addr
}
