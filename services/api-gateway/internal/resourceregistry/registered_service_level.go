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
		ObjectType: constants.ObjectTypeServiceLevel,
		Load:       resourceloaders.LoadServiceLevels,
		Subs: []resourcekit.SubField{
			// owner: no fetch — projects the Owner shell from the SL's
			// account_id stashed in LoadMeta. type=system or type=account.
			{
				Key:      "owner",
				Populate: populateOwnerOnServiceLevel,
			},
			// owner.account: real fetch via the Account loader; writes the
			// full Account into the Owner shell built by the "owner" sub.
			{
				Key:         "owner.account",
				Target:      constants.ObjectTypeAccount,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractOwnerAccountIDFromServiceLevel,
				Populate:    populateOwnerAccountOnServiceLevel,
			},
		},
	})
}

func populateOwnerOnServiceLevel(ctx context.Context, parent any, _ map[string]any) {
	sl := parent.(*apiresource.ServiceLevel)
	accountID, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeServiceLevel, sl.ID, "owner_account_id")
	sl.Owner = buildOwnerShell(accountID)
}

func extractOwnerAccountIDFromServiceLevel(ctx context.Context, parent any) []string {
	sl := parent.(*apiresource.ServiceLevel)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeServiceLevel, sl.ID, "owner_account_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateOwnerAccountOnServiceLevel(ctx context.Context, parent any, loaded map[string]any) {
	sl := parent.(*apiresource.ServiceLevel)
	if sl.Owner == nil {
		return
	}
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeServiceLevel, sl.ID, "owner_account_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		sl.Owner.Account = v.(*apiresource.Account)
	}
}
