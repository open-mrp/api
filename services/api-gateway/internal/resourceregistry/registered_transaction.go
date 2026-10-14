package resourceregistry

import (
	"context"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeTransaction,
		Load:       resourceloaders.LoadTransactions,
		Subs: []include.SubField{
			{
				Key:         "customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractCustomerIDFromTransaction,
				Populate:    populateCustomerOnTransaction,
			},
			{
				Key:         "responsible_user",
				Target:      constants.ObjectTypeAccountUser,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractResponsibleUserIDFromTransaction,
				Populate:    populateResponsibleUserOnTransaction,
			},
			{Key: "allocations", Cardinality: include.CardinalityList, Target: constants.ObjectTypeTransactionAllocation, ExtractRefs: extractAllocationRefsFromTransaction, Populate: populateAllocationsOnTransaction},
		},
	})
	// The transactions LIST returns TransactionSummary (a distinct resource), so it needs its own definition — the detail's customer funcs cast to *TransactionDetail.
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeTransactionSummary,
		// transaction_summary only ever appears as a top-level list root, never as an include target, so Load is never invoked; reuse the transaction loader to satisfy the registry's non-nil Load requirement.
		Load: resourceloaders.LoadTransactions,
		Subs: []include.SubField{
			{
				Key:         "customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractCustomerIDFromTransactionSummary,
				Populate:    populateCustomerOnTransactionSummary,
			},
		},
	})
}

func extractCustomerIDFromTransactionSummary(ctx context.Context, parent any) []string {
	tx := parent.(*apiresource.TransactionSummary)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeTransactionSummary, tx.ID, "customer_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateCustomerOnTransactionSummary(ctx context.Context, parent any, loaded map[string]any) {
	tx := parent.(*apiresource.TransactionSummary)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeTransactionSummary, tx.ID, "customer_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		tx.Customer = v.(*apiresource.Customer)
	}
}

func extractCustomerIDFromTransaction(ctx context.Context, parent any) []string {
	tx := parent.(*apiresource.TransactionDetail)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeTransaction, tx.ID, "customer_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateCustomerOnTransaction(ctx context.Context, parent any, loaded map[string]any) {
	tx := parent.(*apiresource.TransactionDetail)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeTransaction, tx.ID, "customer_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		tx.Customer = v.(*apiresource.Customer)
	}
}

func extractResponsibleUserIDFromTransaction(ctx context.Context, parent any) []string {
	tx := parent.(*apiresource.TransactionDetail)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeTransaction, tx.ID, "responsible_user_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateResponsibleUserOnTransaction(ctx context.Context, parent any, loaded map[string]any) {
	tx := parent.(*apiresource.TransactionDetail)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeTransaction, tx.ID, "responsible_user_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		tx.ResponsibleUser = v.(*apiresource.AccountUser)
	}
}

func populateAllocationsOnTransaction(ctx context.Context, parent any, _ map[string]any) {
	tx := parent.(*apiresource.TransactionDetail)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeTransaction, tx.ID, "allocations")
	if !ok {
		return
	}
	tx.Allocations = v.(*apiresource.List[apiresource.TransactionAllocation])
}

// The resolver runs Populate before gathering refs, so the allocations are already on the transaction.
func extractAllocationRefsFromTransaction(_ context.Context, parent any) []any {
	tx := parent.(*apiresource.TransactionDetail)
	if tx.Allocations == nil {
		return nil
	}
	refs := make([]any, len(tx.Allocations.Data))
	for i := range tx.Allocations.Data {
		refs[i] = &tx.Allocations.Data[i]
	}
	return refs
}
