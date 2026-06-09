package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeTransaction,
		Load:       resourceloaders.LoadTransactions,
		Subs: []resourcekit.SubField{
			{
				Key:         "customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractCustomerIDFromTransaction,
				Populate:    populateCustomerOnTransaction,
			},
			{Key: "responsible_user", Populate: populateResponsibleUserOnTransaction},
			{Key: "allocations", Cardinality: resourcekit.CardinalityList, Populate: populateAllocationsOnTransaction},
		},
	})
	// The transactions LIST returns TransactionSummary (a distinct resource), so it
	// needs its own definition — the detail's customer funcs cast to *TransactionDetail.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeTransactionSummary,
		// transaction_summary only ever appears as a top-level list root, never as
		// an include target, so Load is never invoked; reuse the transaction loader
		// to satisfy the registry's non-nil Load requirement.
		Load: resourceloaders.LoadTransactions,
		Subs: []resourcekit.SubField{
			{
				Key:         "customer",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractCustomerIDFromTransactionSummary,
				Populate:    populateCustomerOnTransactionSummary,
			},
		},
	})
}

func extractCustomerIDFromTransactionSummary(ctx context.Context, parent any) []string {
	tx := parent.(*apiresource.TransactionSummary)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeTransactionSummary, tx.ID, "customer_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateCustomerOnTransactionSummary(ctx context.Context, parent any, loaded map[string]any) {
	tx := parent.(*apiresource.TransactionSummary)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeTransactionSummary, tx.ID, "customer_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		tx.Customer = v.(*apiresource.Customer)
	}
}

func extractCustomerIDFromTransaction(ctx context.Context, parent any) []string {
	tx := parent.(*apiresource.TransactionDetail)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeTransaction, tx.ID, "customer_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateCustomerOnTransaction(ctx context.Context, parent any, loaded map[string]any) {
	tx := parent.(*apiresource.TransactionDetail)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeTransaction, tx.ID, "customer_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		tx.Customer = v.(*apiresource.Customer)
	}
}

func populateResponsibleUserOnTransaction(ctx context.Context, parent any, _ map[string]any) {
	tx := parent.(*apiresource.TransactionDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeTransaction, tx.ID, "responsible_user")
	if !ok {
		return
	}
	tx.ResponsibleUser = v.(*apiresource.AccountUser)
}

func populateAllocationsOnTransaction(ctx context.Context, parent any, _ map[string]any) {
	tx := parent.(*apiresource.TransactionDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeTransaction, tx.ID, "allocations")
	if !ok {
		return
	}
	tx.Allocations = v.(*apiresource.List[apiresource.TransactionAllocation])
}
