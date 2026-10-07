// Package resourceregistry contains the init()-time resourcekit.Definition registrations for every resource the api-gateway resolves includes against. Importing the package (typically via blank-import from cmd/run.go) is what causes the registrations to fire.
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
		ObjectType: constants.ObjectTypeAccount,
		Load:       resourceloaders.LoadAccounts,
		Subs: []resourcekit.SubField{
			{Key: "branding", Populate: populateBrandingOnAccount},
			{Key: "portal", Populate: populatePortalOnAccount},
			{Key: "default_billing_address", Target: constants.ObjectTypeAddress, Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs: accountAddressIDs("default_billing_address_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					a := parent.(*apiresource.Account)
					a.DefaultBillingAddress = loadedAccountAddress(ctx, a.ID, "default_billing_address_id", loaded)
				}},
			{Key: "default_shipping_address", Target: constants.ObjectTypeAddress, Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs: accountAddressIDs("default_shipping_address_id"),
				Populate: func(ctx context.Context, parent any, loaded map[string]any) {
					a := parent.(*apiresource.Account)
					a.DefaultShippingAddress = loadedAccountAddress(ctx, a.ID, "default_shipping_address_id", loaded)
				}},
		},
	})
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypePublicAccount,
		Load:       resourceloaders.LoadPublicAccounts,
	})
}

func populateBrandingOnAccount(ctx context.Context, parent any, _ map[string]any) {
	a := parent.(*apiresource.Account)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAccount, a.ID, "branding")
	if !ok {
		return
	}
	a.Branding = v.(*apiresource.AccountBranding)
}

func populatePortalOnAccount(ctx context.Context, parent any, _ map[string]any) {
	a := parent.(*apiresource.Account)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAccount, a.ID, "portal")
	if !ok {
		return
	}
	a.Portal = v.(*apiresource.AccountPortal)
}

func accountAddressIDs(key string) func(ctx context.Context, parent any) []string {
	return func(ctx context.Context, parent any) []string {
		id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeAccount, parent.(*apiresource.Account).ID, key)
		if id == "" {
			return nil
		}
		return []string{id}
	}
}

func loadedAccountAddress(ctx context.Context, accountID, key string, loaded map[string]any) *apiresource.Address {
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeAccount, accountID, key)
	if v, ok := loaded[id].(*apiresource.Address); ok {
		return v
	}
	return nil
}
