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
		ObjectType: constants.ObjectTypeUnit,
		Load:       resourceloaders.LoadUnits,
		Subs: []resourcekit.SubField{
			{Key: "owner", Populate: populateOwnerOnUnit},
			{
				Key:         "owner.account",
				Target:      constants.ObjectTypeAccount,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractOwnerAccountIDFromUnit,
				Populate:    populateOwnerAccountOnUnit,
			},
		},
	})
}

func populateOwnerOnUnit(ctx context.Context, parent any, _ map[string]any) {
	u := parent.(*apiresource.Unit)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeUnit, u.ID, "owner_account_id")
	u.Owner = buildOwnerShell(id)
}

func extractOwnerAccountIDFromUnit(ctx context.Context, parent any) []string {
	u := parent.(*apiresource.Unit)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeUnit, u.ID, "owner_account_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateOwnerAccountOnUnit(ctx context.Context, parent any, loaded map[string]any) {
	u := parent.(*apiresource.Unit)
	if u.Owner == nil {
		return
	}
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeUnit, u.ID, "owner_account_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		u.Owner.Account = v.(*apiresource.Account)
	}
}
