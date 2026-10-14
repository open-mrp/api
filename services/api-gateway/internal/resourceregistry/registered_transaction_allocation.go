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
		ObjectType: constants.ObjectTypeTransactionAllocation,
		Load:       resourceloaders.LoadTransactionAllocations,
		Subs: []include.SubField{
			{
				// The amount is already on the allocation — this exists so a caller can reach through
				// it to the currency it is counted in.
				Key:         "amount",
				Target:      constants.ObjectTypeQuantity,
				Cardinality: include.CardinalityOnePtr,
				ExtractRefs: extractAmountRefFromTransactionAllocation,
			},
			{
				Key:         "transaction",
				Target:      constants.ObjectTypeTransaction,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractTransactionIDFromTransactionAllocation,
				Populate:    populateTransactionOnTransactionAllocation,
			},
		},
	})
}

func extractTransactionIDFromTransactionAllocation(ctx context.Context, parent any) []string {
	a := parent.(*apiresource.TransactionAllocation)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeTransactionAllocation, a.ID, "transaction_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateTransactionOnTransactionAllocation(ctx context.Context, parent any, loaded map[string]any) {
	a := parent.(*apiresource.TransactionAllocation)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeTransactionAllocation, a.ID, "transaction_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		a.Transaction = v.(*apiresource.TransactionDetail)
	}
}

func extractAmountRefFromTransactionAllocation(_ context.Context, parent any) []any {
	a := parent.(*apiresource.TransactionAllocation)
	if a.Amount == nil {
		return nil
	}
	return []any{a.Amount}
}
