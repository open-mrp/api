package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeAccountPrice,
		Load:       resourceloaders.LoadAccountPrices,
		Subs: []resourcekit.SubField{
			{
				Key:         "recipient_account",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractRecipientAccountIDFromAccountPrice,
				Populate:    populateRecipientAccountOnAccountPrice,
			},
			{Key: "product_line", Populate: populateProductLineOnAccountPrice},
			{Key: "categories", Populate: populateCategoriesOnAccountPrice},
			{Key: "attributes", Populate: populateAttributesOnAccountPrice},
		},
	})
}

func extractRecipientAccountIDFromAccountPrice(ctx context.Context, parent any) []string {
	ap := parent.(*apiresource.AccountPrice)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountPrice, ap.ID, "recipient_account_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateRecipientAccountOnAccountPrice(ctx context.Context, parent any, loaded map[string]any) {
	ap := parent.(*apiresource.AccountPrice)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountPrice, ap.ID, "recipient_account_id")
	if v, ok := loaded[id]; ok {
		ap.RecipientAccount = v.(*apiresource.Customer)
	}
}

func populateProductLineOnAccountPrice(ctx context.Context, parent any, _ map[string]any) {
	ap := parent.(*apiresource.AccountPrice)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAccountPrice, ap.ID, "product_line")
	if !ok {
		return
	}
	ap.ProductLine = v.(*apiresource.ProductLine)
}

func populateCategoriesOnAccountPrice(ctx context.Context, parent any, _ map[string]any) {
	ap := parent.(*apiresource.AccountPrice)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAccountPrice, ap.ID, "categories")
	if !ok {
		return
	}
	ap.Categories = v.(*apiresource.List[apiresource.ItemCategory])
}

func populateAttributesOnAccountPrice(ctx context.Context, parent any, _ map[string]any) {
	ap := parent.(*apiresource.AccountPrice)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAccountPrice, ap.ID, "attributes")
	if !ok {
		return
	}
	ap.Attributes = v.(*apiresource.List[apiresource.Attribute])
}
