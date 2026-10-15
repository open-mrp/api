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
		ObjectType: constants.ObjectTypeServiceLevel,
		Load:       resourceloaders.LoadServiceLevels,
		Subs: []include.SubField{
			// owner: no fetch — projects the Owner shell from the SL's account_id stashed in LoadMeta. type=system or type=account.
			{
				Key:      "owner",
				Populate: populateOwnerOnServiceLevel,
			},
			// owner.account: real fetch via the Account loader; writes the full Account into the Owner shell built by the "owner" sub.
			{
				Key:         "owner.account",
				Target:      constants.ObjectTypeAccount,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractOwnerAccountIDFromServiceLevel,
				Populate:    populateOwnerAccountOnServiceLevel,
			},
			{
				Key:         "carrier",
				Target:      constants.ObjectTypeCarrier,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractCarrierIDFromServiceLevel,
				Populate:    populateCarrierOnServiceLevel,
			},
		},
	})
}

func populateOwnerOnServiceLevel(ctx context.Context, parent any, _ map[string]any) {
	sl := parent.(*apiresource.ServiceLevel)
	accountID, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeServiceLevel, sl.ID, "owner_account_id")
	sl.Owner = buildOwnerShell(accountID)
}

func extractOwnerAccountIDFromServiceLevel(ctx context.Context, parent any) []string {
	sl := parent.(*apiresource.ServiceLevel)
	id, _ := include.GetLoadMeta(ctx).
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
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeServiceLevel, sl.ID, "owner_account_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		sl.Owner.Account = v.(*apiresource.Account)
	}
}

func extractCarrierIDFromServiceLevel(ctx context.Context, parent any) []string {
	sl := parent.(*apiresource.ServiceLevel)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeServiceLevel, sl.ID, "carrier_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateCarrierOnServiceLevel(ctx context.Context, parent any, loaded map[string]any) {
	sl := parent.(*apiresource.ServiceLevel)
	id, _ := include.GetLoadMeta(ctx).GetString(constants.ObjectTypeServiceLevel, sl.ID, "carrier_id")
	if v, ok := loaded[id]; ok && id != "" {
		sl.Carrier = v.(*apiresource.Carrier)
	}
}
