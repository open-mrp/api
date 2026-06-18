// Package resourceregistry contains the init()-time resourcekit.Definition registrations for every resource the api-gateway resolves includes against. Importing the package (typically via blank-import from cmd/run.go) is what causes the registrations to fire.
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
		ObjectType: constants.ObjectTypeAccount,
		Load:       resourceloaders.LoadAccounts,
		Subs: []resourcekit.SubField{
			{Key: "branding", Populate: populateBrandingOnAccount},
			{Key: "portal", Populate: populatePortalOnAccount},
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
