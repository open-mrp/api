package grpc

import (
	"context"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (h *gRPCHandler) ListInvoices(ctx context.Context, req *pb.ListInvoicesRequest) (*pb.ListInvoicesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListInvoicesParams{
		Cursor:           req.Cursor,
		Limit:            req.Limit,
		Query:            req.Query,
		Status:           req.Status,
		ItemIDs:          req.ItemIds,
		CustomerIDs:      req.CustomerIds,
		ProductLineIDs:   req.ProductLineIds,
		CustomerGroupIDs: req.CustomerGroupIds,
		SalesRepIDs:      req.SalesRepIds,
		Includes:         req.Includes,
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		params.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		params.EndDate = &t
	}

	result, apiErr := h.invoiceSvc.ListInvoices(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	invoices := make([]*pb.InvoiceInfo, len(result.Invoices))
	for i, inv := range result.Invoices {
		invoices[i] = invoiceToProto(inv)
	}

	return &pb.ListInvoicesResponse{
		Invoices: invoices,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *gRPCHandler) GetInvoice(ctx context.Context, req *pb.GetInvoiceRequest) (*pb.GetInvoiceResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	invoice, apiErr := h.invoiceSvc.GetInvoice(ctx, domain.GetInvoiceParams{
		InvoiceID: req.Id,
		Includes:  req.Includes,
	})
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetInvoiceResponse{
		Invoice: invoiceToProto(invoice),
	}, nil
}

func (h *gRPCHandler) UpdateInvoice(ctx context.Context, req *pb.UpdateInvoiceRequest) (*pb.UpdateInvoiceResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ctx, finalizeIdempotency := contracts.WithIdempotencyTracking(ctx)
	defer finalizeIdempotency()

	params := domain.UpdateInvoiceParams{
		InvoiceID: req.Id,
		Includes:  req.Includes,
		Note:      field.StringClearableFromProto(req.Note),
	}
	if req.HasBeenSent != nil {
		params.HasBeenSent = req.HasBeenSent
	}
	if req.IsEdiSent != nil {
		params.IsEdiSent = req.IsEdiSent
	}
	if req.IsPaidInFull != nil {
		params.IsPaidInFull = req.IsPaidInFull
	}

	result, apiErr := h.invoiceSvc.UpdateInvoice(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.UpdateInvoiceResponse{
		Invoice: invoiceToProto(result),
	}, nil
}

func (h *gRPCHandler) ListCustomerInvoices(ctx context.Context, req *pb.ListCustomerInvoicesRequest) (*pb.ListCustomerInvoicesResponse, error) {
	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	params := domain.ListCustomerInvoicesParams{
		CustomerAccountID: req.CustomerAccountId,
		Cursor:            req.Cursor,
		Limit:             req.Limit,
		Query:             req.Query,
		Includes:          req.Includes,
	}

	result, apiErr := h.invoiceSvc.ListCustomerInvoices(ctx, params)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	invoices := make([]*pb.InvoiceForPaymentInfo, len(result.Invoices))
	for i, inv := range result.Invoices {
		invoices[i] = invoiceForPaymentToProto(inv)
	}

	return &pb.ListCustomerInvoicesResponse{
		Invoices: invoices,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

// Proto conversion helpers

func invoiceToProto(inv *domain.Invoice) *pb.InvoiceInfo {
	if inv == nil {
		return nil
	}

	lines := make([]*pb.InvoiceLineInfo, len(inv.Lines))
	for i, l := range inv.Lines {
		lines[i] = invoiceLineToProto(l)
	}

	allocations := make([]*pb.InvoiceAllocationInfo, len(inv.Allocations))
	for i, a := range inv.Allocations {
		allocations[i] = invoiceAllocationToProto(a)
	}

	info := &pb.InvoiceInfo{
		Id:                       inv.ID,
		Number:                   inv.Number,
		Note:                     inv.Note,
		OrderId:                  inv.OrderID,
		OrderNumber:              inv.OrderNumber,
		BillingAddressId:         inv.BillingAddressID,
		BillingAddressName:       inv.BillingAddressName,
		BillingAddressLine1:      inv.BillingAddressLine1,
		BillingAddressLine2:      inv.BillingAddressLine2,
		BillingAddressCity:       inv.BillingAddressCity,
		BillingAddressState:      inv.BillingAddressState,
		BillingAddressZip:        inv.BillingAddressZip,
		BillingAddressCountry:    inv.BillingAddressCountry,
		ShipmentId:               inv.ShipmentID,
		ShipmentNumber:           inv.ShipmentNumber,
		IsPaidInFull:             inv.IsPaidInFull,
		IsOverPaid:               inv.IsOverPaid,
		IsEdiSent:                inv.IsEdiSent,
		HasBeenSent:              inv.HasBeenSent,
		AcceptsInvoiceEmails:     inv.AcceptsInvoiceEmails,
		Lines:                    lines,
		Allocations:              allocations,
		CustomerId:               inv.CustomerID,
		CustomerName:             inv.CustomerName,
		CustomerNumber:           inv.CustomerNumber,
		CustomerIsEdiEnabled:     inv.CustomerIsEdiEnabled,
		PriorityCode:             string(inv.PriorityCode),
		PaymentTermId:            inv.PaymentTermID,
		PaymentTermName:          inv.PaymentTermName,
		PaymentTermIsActive:      inv.PaymentTermIsActive,
		LineCount:                inv.LineCount,
		TotalInvoiced:            inv.TotalInvoiced,
		CustomerStatusCode:       inv.CustomerStatusCode,
		CustomerCommissionPolicy: inv.CustomerCommissionPolicy,
		CreatedAt:                timestamppb.New(inv.CreatedAt),
		UpdatedAt:                timestamppb.New(inv.UpdatedAt),
	}

	return info
}

func invoiceLineToProto(l *domain.InvoiceLine) *pb.InvoiceLineInfo {
	if l == nil {
		return nil
	}

	info := &pb.InvoiceLineInfo{
		Id:                         l.ID,
		QuantityId:                 l.QuantityID,
		QuantityValue:              l.QuantityValue,
		QuantityUnitId:             l.QuantityUnitID,
		QuantityUnitAbbreviation:   l.QuantityUnitAbbr,
		UnitPriceId:                l.UnitPriceID,
		UnitPriceValue:             l.UnitPriceValue,
		UnitPriceNumeratorUnitId:   l.UnitPriceNumUnit,
		UnitPriceDenominatorUnitId: l.UnitPriceDenUnit,
		OrderLineId:                l.OrderLineID,
		OrderLineItemId:            l.OrderLineItemID,
		OrderLineItemSku:           l.OrderLineItemSKU,
		OrderLineProductId:         l.OrderLineProductID,
		CreatedAt:                  timestamppb.New(l.CreatedAt),
		UpdatedAt:                  timestamppb.New(l.UpdatedAt),
	}

	return info
}

func invoiceAllocationToProto(a *domain.InvoiceAllocation) *pb.InvoiceAllocationInfo {
	if a == nil {
		return nil
	}

	return &pb.InvoiceAllocationInfo{
		Id:                     a.ID,
		TransactionId:          a.TransactionID,
		AmountId:               a.AmountID,
		AmountValue:            a.AmountValue,
		AmountUnitId:           a.AmountUnitID,
		AmountUnitAbbreviation: a.AmountUnitAbbr,
		Note:                   a.Note,
		CreatedAt:              timestamppb.New(a.CreatedAt),
		UpdatedAt:              timestamppb.New(a.UpdatedAt),
	}
}

func invoiceForPaymentToProto(inv *domain.InvoiceForPayment) *pb.InvoiceForPaymentInfo {
	if inv == nil {
		return nil
	}

	allocations := make([]*pb.InvoiceAllocationInfo, len(inv.Allocations))
	for i, a := range inv.Allocations {
		allocations[i] = invoiceAllocationToProto(a)
	}

	return &pb.InvoiceForPaymentInfo{
		Id:                 inv.ID,
		Number:             inv.Number,
		CustomerPo:         inv.CustomerPO,
		CustomerId:         inv.CustomerID,
		CustomerName:       inv.CustomerName,
		CustomerNumber:     inv.CustomerNumber,
		IsParentAccount:    inv.IsParentAccount,
		ParentAccountId:    inv.ParentAccountID,
		IsPrepaid:          inv.IsPrepaid,
		BillingAddressId:   inv.BillingAddressID,
		BillingAddressName: inv.BillingAddressName,
		InvoiceTotal:       inv.InvoiceTotal,
		IsPaidInFull:       inv.IsPaidInFull,
		Allocations:        allocations,
		CreatedAt:          timestamppb.New(inv.CreatedAt),
		UpdatedAt:          timestamppb.New(inv.UpdatedAt),
	}
}
