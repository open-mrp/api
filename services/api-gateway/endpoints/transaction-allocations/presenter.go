package transactionallocationep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func AllocationEntryPresenter(d *pb.AllocationEntryInfo) apiresource.AllocationEntry {
	if d == nil {
		return apiresource.AllocationEntry{}
	}

	return apiresource.AllocationEntry{
		ID:            d.Id,
		Object:        constants.ObjectTypeAllocationEntry,
		Amount:        d.AmountValue,
		DisplayAmount: apiresource.FormatDisplayValue(d.AmountValue, d.AmountUnitAbbr, string(constants.UnitTypeCurrency)),
		Customer: &apiresource.AllocationCustomer{
			Name:   d.CustomerName,
			Number: d.CustomerNumber,
		},
		Transaction: &apiresource.AllocationTransaction{
			ID:             d.TransactionId,
			Object:         constants.ObjectTypeTransaction,
			Type:           d.TransactionType,
			Method:         d.TransactionMethod,
			AdjustmentType: d.AdjustmentType,
		},
		Invoice: &apiresource.AllocationInvoice{
			ID:     d.InvoiceId,
			Object: constants.ObjectTypeInvoiceSummary,
			Number: d.InvoiceNumber,
		},
		Note:      d.Note,
		CreatedAt: grpcutil.TimestampToTime(d.CreatedAt),
	}
}

func AllocationEntryListPresenter(resp *pb.ListAllocationEntriesResponse) *apiresource.List[apiresource.AllocationEntry] {
	if resp == nil {
		return apiresource.NewList[apiresource.AllocationEntry](nil, apiresource.PageInfo{})
	}

	entries := make([]apiresource.AllocationEntry, len(resp.Entries))
	for i, d := range resp.Entries {
		entries[i] = AllocationEntryPresenter(d)
	}

	return apiresource.NewList(entries, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func TransactionAllocationPresenter(a *pb.TransactionAllocationInfo) apiresource.TransactionAllocation {
	if a == nil {
		return apiresource.TransactionAllocation{}
	}

	alloc := apiresource.TransactionAllocation{
		ID:     a.Id,
		Object: constants.ObjectTypeTransactionAllocation,
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
		Note: a.Note,
		Transaction: &apiresource.TransactionDetail{
			ID:     a.TransactionId,
			Object: constants.ObjectTypeTransaction,
		},
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}

	if a.InvoiceId != nil && *a.InvoiceId != "" {
		invoiceNumber := ""
		if a.InvoiceNumber != nil {
			invoiceNumber = *a.InvoiceNumber
		}
		alloc.Invoice = &apiresource.InvoiceSummary{
			ID:     *a.InvoiceId,
			Object: constants.ObjectTypeInvoiceSummary,
			Number: invoiceNumber,
		}
	}

	return alloc
}

func OpenCreditEntryPresenter(d *pb.OpenCreditEntryInfo) apiresource.OpenCreditEntry {
	if d == nil {
		return apiresource.OpenCreditEntry{}
	}

	allocations := make([]apiresource.InvoiceAllocationEntry, len(d.InvoiceAllocations))
	for i, a := range d.InvoiceAllocations {
		allocations[i] = apiresource.InvoiceAllocationEntry{
			InvoiceNumber: a.InvoiceNumber,
			Amount:        a.Amount,
		}
	}

	return apiresource.OpenCreditEntry{
		ID:              d.Id,
		Object:          constants.ObjectTypeOpenCreditEntry,
		Number:          d.Number,
		OriginalAmount:  d.OriginalAmount,
		AllocatedAmount: d.AllocatedAmount,
		LeftoverAmount:  d.LeftoverAmount,
		Customer: &apiresource.AllocationCustomer{
			Name:   d.CustomerName,
			Number: d.CustomerNumber,
		},
		TransactionType:     d.TransactionType,
		TransactionMethod:   d.TransactionMethod,
		AdjustmentType:      d.AdjustmentType,
		ResponsibleUserName: d.ResponsibleUserName,
		Note:                d.Note,
		StripePaymentID:     d.StripePaymentId,
		InvoiceAllocations:  allocations,
		CreatedAt:           grpcutil.TimestampToTime(d.CreatedAt),
	}
}

func OpenCreditListPresenter(resp *pb.ListOpenCreditsResponse) *apiresource.List[apiresource.OpenCreditEntry] {
	if resp == nil {
		return apiresource.NewList[apiresource.OpenCreditEntry](nil, apiresource.PageInfo{})
	}

	entries := make([]apiresource.OpenCreditEntry, len(resp.Entries))
	for i, d := range resp.Entries {
		entries[i] = OpenCreditEntryPresenter(d)
	}

	var pi apiresource.PageInfo
	if resp.PageInfo != nil {
		pi = grpcutil.MapProtoPageInfo(resp.PageInfo)
	}

	return apiresource.NewList(entries, pi)
}
