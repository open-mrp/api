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
		ObjectType: constants.ObjectTypeAccountPrice,
		Load:       resourceloaders.LoadAccountPrices,
		Subs: []include.SubField{
			{
				Key:         "recipient_account",
				Target:      constants.ObjectTypeCustomer,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractRecipientAccountIDFromAccountPrice,
				Populate:    populateRecipientAccountOnAccountPrice,
			},
			{
				Key:         "product_line",
				Target:      constants.ObjectTypeProductLine,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractProductLineIDFromAccountPrice,
				Populate:    populateProductLineOnAccountPrice,
			},
			{Key: "categories", Populate: populateCategoriesOnAccountPrice},
			{Key: "attributes", Populate: populateAttributesOnAccountPrice},
		},
	})
}

func extractRecipientAccountIDFromAccountPrice(ctx context.Context, parent any) []string {
	ap := parent.(*apiresource.AccountPrice)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountPrice, ap.ID, "recipient_account_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateRecipientAccountOnAccountPrice(ctx context.Context, parent any, loaded map[string]any) {
	ap := parent.(*apiresource.AccountPrice)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountPrice, ap.ID, "recipient_account_id")
	if v, ok := loaded[id]; ok {
		ap.RecipientAccount = v.(*apiresource.Customer)
	}
}

func extractProductLineIDFromAccountPrice(ctx context.Context, parent any) []string {
	ap := parent.(*apiresource.AccountPrice)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountPrice, ap.ID, "product_line_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateProductLineOnAccountPrice(ctx context.Context, parent any, loaded map[string]any) {
	ap := parent.(*apiresource.AccountPrice)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountPrice, ap.ID, "product_line_id")
	if v, ok := loaded[id]; ok {
		ap.ProductLine = v.(*apiresource.ProductLine)
	}
}

func populateCategoriesOnAccountPrice(ctx context.Context, parent any, _ map[string]any) {
	ap := parent.(*apiresource.AccountPrice)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAccountPrice, ap.ID, "categories")
	if !ok {
		return
	}
	ap.Categories = v.(*apiresource.List[apiresource.ItemCategory])
}

func populateAttributesOnAccountPrice(ctx context.Context, parent any, _ map[string]any) {
	ap := parent.(*apiresource.AccountPrice)
	v, ok := include.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAccountPrice, ap.ID, "attributes")
	if !ok {
		return
	}
	ap.Attributes = v.(*apiresource.List[apiresource.Attribute])
}
