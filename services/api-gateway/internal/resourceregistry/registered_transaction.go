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
			{Key: "customer", Populate: populateCustomerOnTransaction},
			{Key: "responsible_user", Populate: populateResponsibleUserOnTransaction},
			{Key: "allocations", Cardinality: resourcekit.CardinalityList, Populate: populateAllocationsOnTransaction},
		},
	})
}

func populateCustomerOnTransaction(ctx context.Context, parent any, _ map[string]any) {
	tx := parent.(*apiresource.TransactionDetail)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeTransaction, tx.ID, "customer")
	if !ok {
		return
	}
	tx.Customer = v.(*apiresource.Customer)
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
