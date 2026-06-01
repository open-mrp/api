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
		ObjectType: constants.ObjectTypePaymentTerm,
		Load:       resourceloaders.LoadPaymentTerms,
		Subs: []resourcekit.SubField{
			{Key: "owner", Populate: populateOwnerOnPaymentTerm},
			{
				Key:         "owner.account",
				Target:      constants.ObjectTypeAccount,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractOwnerAccountIDFromPaymentTerm,
				Populate:    populateOwnerAccountOnPaymentTerm,
			},
		},
	})
}

func populateOwnerOnPaymentTerm(ctx context.Context, parent any, _ map[string]any) {
	pt := parent.(*apiresource.PaymentTerm)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypePaymentTerm, pt.ID, "owner_account_id")
	pt.Owner = buildOwnerShell(id)
}

func extractOwnerAccountIDFromPaymentTerm(ctx context.Context, parent any) []string {
	pt := parent.(*apiresource.PaymentTerm)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypePaymentTerm, pt.ID, "owner_account_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateOwnerAccountOnPaymentTerm(ctx context.Context, parent any, loaded map[string]any) {
	pt := parent.(*apiresource.PaymentTerm)
	if pt.Owner == nil {
		return
	}
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypePaymentTerm, pt.ID, "owner_account_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		pt.Owner.Account = v.(*apiresource.Account)
	}
}
