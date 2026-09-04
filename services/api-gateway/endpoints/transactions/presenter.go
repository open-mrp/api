package transactionep

import (
	"context"

	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
)

// TransactionDetailPresenter builds the transaction detail resource and
// stashes the expandable sub-resource data (customer FK, responsible_user FK,
// allocations) in the request-scoped LoadMeta so the include resolver can
// populate them when requested. Expandable fields stay nil here — never
// fabricated.
func TransactionDetailPresenter(ctx context.Context, d *pb.TransactionInfo) apiresource.TransactionDetail {
	if d == nil {
		return apiresource.TransactionDetail{}
	}

	createdAt := grpcutil.TimestampToTime(d.CreatedAt)
	updatedAt := grpcutil.TimestampToTime(d.UpdatedAt)
	meta := resourcekit.GetLoadMeta(ctx)

	tx := apiresource.TransactionDetail{
		ID:               d.Id,
		Object:           constants.ObjectTypeTransaction,
		Number:           d.Number,
		IsFullyAllocated: d.IsFullyAllocated,
		StripePaymentID:  d.StripePaymentId,
		AllocationCount:  d.AllocationCount,
		Note:             d.Note,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}

	tx.Amount = &apiresource.Quantity{
		ID:           d.AmountId,
		Object:       constants.ObjectTypeQuantity,
		Value:        d.AmountValue,
		DisplayValue: apiresource.FormatDisplayValue(d.AmountValue, d.AmountUnitAbbreviation, string(constants.UnitTypeCurrency)),
		// Unit left nil: its id is stashed so `amount.unit` resolves the real unit; never fabricated.
	}
	meta.Set(constants.ObjectTypeQuantity, d.AmountId, "unit_id", d.AmountUnitId)

	tx.TransactionType = &apiresource.TransactionType{
		ID:     d.TransactionTypeId,
		Object: constants.ObjectTypeTransactionType,
		Name:   d.TransactionTypeName,
		Code:   constants.TransactionType(d.TransactionTypeCode),
	}

	if d.TransactionMethodId != nil {
		tx.TransactionMethod = &apiresource.TransactionMethod{
			ID:     *d.TransactionMethodId,
			Object: constants.ObjectTypeTransactionMethod,
		}
		if d.TransactionMethodName != nil {
			tx.TransactionMethod.Name = *d.TransactionMethodName
		}
		if d.TransactionMethodCode != nil {
			tx.TransactionMethod.Code = constants.TransactionMethod(*d.TransactionMethodCode)
		}
	}

	if d.AdjustmentTypeId != nil {
		// Owner left nil: expandable, and not derivable here (adjustment
		// types may be system- or account-owned). Never fabricate.
		tx.AdjustmentType = &apiresource.AdjustmentType{
			ID:        *d.AdjustmentTypeId,
			Object:    constants.ObjectTypeAdjustmentType,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
		if d.AdjustmentTypeName != nil {
			tx.AdjustmentType.Name = *d.AdjustmentTypeName
		}
		if d.AdjustmentTypeCode != nil {
			tx.AdjustmentType.Code = constants.AdjustmentType(*d.AdjustmentTypeCode)
		}
	}

	if d.CustomerId != nil && *d.CustomerId != "" {
		meta.Set(constants.ObjectTypeTransaction, tx.ID, "customer_id", *d.CustomerId)
	}

	if d.ResponsibleUserId != nil && *d.ResponsibleUserId != "" {
		meta.Set(constants.ObjectTypeTransaction, tx.ID, "responsible_user_id", *d.ResponsibleUserId)
	}

	if d.Allocations != nil {
		allocations := make([]apiresource.TransactionAllocation, len(d.Allocations))
		for i, a := range d.Allocations {
			allocations[i] = TransactionAllocationPresenter(meta, a)
		}
		meta.Set(constants.ObjectTypeTransaction, tx.ID, "allocations",
			apiresource.NewList(allocations, apiresource.PageInfo{}))
	}

	return tx
}

func TransactionSummaryPresenter(d *pb.TransactionSummaryInfo) apiresource.TransactionSummary {
	if d == nil {
		return apiresource.TransactionSummary{}
	}

	createdAt := grpcutil.TimestampToTime(d.CreatedAt)
	updatedAt := grpcutil.TimestampToTime(d.UpdatedAt)

	ts := apiresource.TransactionSummary{
		ID:               d.Id,
		Object:           constants.ObjectTypeTransactionSummary,
		Number:           d.Number,
		IsFullyAllocated: d.IsFullyAllocated,
		AllocationCount:  d.AllocationCount,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}

	ts.Amount = &apiresource.Quantity{
		ID:           d.AmountId,
		Object:       constants.ObjectTypeQuantity,
		Value:        d.AmountValue,
		DisplayValue: apiresource.FormatDisplayValue(d.AmountValue, d.AmountUnitAbbreviation, string(constants.UnitTypeCurrency)),
		// Unit left nil: the expandable unit loads real data via ?include= and is
		// never fabricated; display_value already carries the formatted amount.
	}

	// customer is an expandable reference loaded with real data via
	// ?include=customer; left nil here rather than fabricated.

	ts.TransactionType = &apiresource.TransactionType{
		ID:     d.TransactionTypeId,
		Object: constants.ObjectTypeTransactionType,
		Name:   d.TransactionTypeName,
		Code:   constants.TransactionType(d.TransactionTypeCode),
	}

	if d.TransactionMethodId != nil {
		ts.TransactionMethod = &apiresource.TransactionMethod{
			ID:     *d.TransactionMethodId,
			Object: constants.ObjectTypeTransactionMethod,
		}
		if d.TransactionMethodName != nil {
			ts.TransactionMethod.Name = *d.TransactionMethodName
		}
		if d.TransactionMethodCode != nil {
			ts.TransactionMethod.Code = constants.TransactionMethod(*d.TransactionMethodCode)
		}
	}

	if d.AdjustmentTypeId != nil {
		ts.AdjustmentType = &apiresource.AdjustmentType{
			ID:        *d.AdjustmentTypeId,
			Object:    constants.ObjectTypeAdjustmentType,
			Owner:     apiresource.SystemOwner(),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		}
		if d.AdjustmentTypeName != nil {
			ts.AdjustmentType.Name = *d.AdjustmentTypeName
		}
		if d.AdjustmentTypeCode != nil {
			ts.AdjustmentType.Code = constants.AdjustmentType(*d.AdjustmentTypeCode)
		}
	}

	return ts
}

func TransactionAllocationPresenter(meta *resourcekit.LoadMeta, a *pb.TransactionAllocationInfo) apiresource.TransactionAllocation {
	if a == nil {
		return apiresource.TransactionAllocation{}
	}

	// The currency the amount is counted in is a record of its own — stashed as an id so
	// `allocations.amount.unit` resolves it in full.
	meta.Set(constants.ObjectTypeQuantity, a.AmountId, "unit_id", a.AmountUnitId)
	meta.Set(constants.ObjectTypeTransactionAllocation, a.Id, "transaction_id", a.TransactionId)

	alloc := apiresource.TransactionAllocation{
		ID:     a.Id,
		Object: constants.ObjectTypeTransactionAllocation,
		Amount: &apiresource.Quantity{
			ID:           a.AmountId,
			Object:       constants.ObjectTypeQuantity,
			Value:        a.AmountValue,
			DisplayValue: apiresource.FormatDisplayValue(a.AmountValue, a.AmountUnitAbbreviation, string(constants.UnitTypeCurrency)),
			// Unit left nil: its id is stashed so `allocations.amount.unit` resolves the real unit; never fabricated.
		},
		Note:      a.Note,
		CreatedAt: grpcutil.TimestampToTime(a.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(a.UpdatedAt),
	}

	return alloc
}

func TransactionListPresenter(ctx context.Context, resp *pb.ListTransactionsResponse) *apiresource.List[apiresource.TransactionSummary] {
	if resp == nil {
		return apiresource.NewList[apiresource.TransactionSummary](nil, apiresource.PageInfo{})
	}

	transactions := make([]apiresource.TransactionSummary, len(resp.Transactions))
	for i, d := range resp.Transactions {
		transactions[i] = TransactionSummaryPresenter(d)
		// customer is an expandable reference: stash the FK id so LoadCustomers
		// fetches the real Customer on ?include=customer. Never fabricate.
		if d.CustomerId != nil && *d.CustomerId != "" {
			resourcekit.GetLoadMeta(ctx).Set(constants.ObjectTypeTransactionSummary, d.Id, "customer_id", *d.CustomerId)
		}
	}

	return apiresource.NewList(transactions, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}

func AccountTransactionListPresenter(ctx context.Context, resp *pb.ListAccountTransactionsResponse) *apiresource.List[apiresource.TransactionDetail] {
	if resp == nil {
		return apiresource.NewList[apiresource.TransactionDetail](nil, apiresource.PageInfo{})
	}

	transactions := make([]apiresource.TransactionDetail, len(resp.Transactions))
	for i, d := range resp.Transactions {
		transactions[i] = TransactionDetailPresenter(ctx, d)
	}

	return apiresource.NewList(transactions, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo))
}
