package settlementep

import (
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/core"
)

func SettlementPresenter(d *pb.SettlementInfo) apiresource.Settlement {
	if d == nil {
		return apiresource.Settlement{}
	}

	settlement := apiresource.Settlement{
		ID:        d.Id,
		Object:    constants.ObjectTypeSettlement,
		Number:    d.Number,
		Note:      d.Note,
		CreatedAt: grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(d.UpdatedAt),
	}

	if d.ResponsibleUserId != nil {
		user := &apiresource.AccountUser{
			ID:     *d.ResponsibleUserId,
			Object: constants.ObjectTypeAccountUser,
		}
		settlement.ResponsibleUser = user
	}

	if d.Allocations != nil {
		allocations := make([]apiresource.TransactionAllocation, len(d.Allocations))
		for i, a := range d.Allocations {
			allocations[i] = TransactionAllocationPresenter(a)
		}
		settlement.Allocations = apiresource.NewList(allocations, apiresource.PageInfo{})
	}

	return settlement
}

func SettlementSummaryPresenter(d *pb.SettlementSummaryInfo) apiresource.SettlementSummary {
	if d == nil {
		return apiresource.SettlementSummary{}
	}

	return apiresource.SettlementSummary{
		ID:               d.Id,
		Object:           constants.ObjectTypeSettlementSummary,
		Number:           d.Number,
		AllocationCount:  d.AllocationCount,
		TotalPayments:    d.TotalPayments,
		TotalRebates:     d.TotalRebates,
		TotalAdjustments: d.TotalAdjustments,
		TotalCredits:     d.TotalCredits,
		InvoiceNumbers:   d.InvoiceNumbers,
		CustomerNames:    d.CustomerNames,
		CreatedAt:        grpcutil.TimestampToTime(d.CreatedAt),
		UpdatedAt:        grpcutil.TimestampToTime(d.UpdatedAt),
	}
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

	invoiceID := ""
	invoiceNumber := ""
	if a.InvoiceId != nil {
		invoiceID = *a.InvoiceId
	}
	if a.InvoiceNumber != nil {
		invoiceNumber = *a.InvoiceNumber
	}
	if invoiceID != "" {
		alloc.Invoice = &apiresource.InvoiceSummary{
			ID:     invoiceID,
			Object: constants.ObjectTypeInvoiceSummary,
			Number: invoiceNumber,
		}
	}

	return alloc
}

func SettlementListPresenter(resp *pb.ListSettlementsResponse) *apiresource.List[apiresource.SettlementSummary] {
	if resp == nil {
		return apiresource.NewList[apiresource.SettlementSummary](nil, apiresource.PageInfo{})
	}

	settlements := make([]apiresource.SettlementSummary, len(resp.Settlements))
	for i, d := range resp.Settlements {
		settlements[i] = SettlementSummaryPresenter(d)
	}

	return apiresource.NewList(settlements, grpcutil.MapProtoPageInfo(resp.PageInfo))
}
