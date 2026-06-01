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
		ObjectType: constants.ObjectTypeRole,
		Load:       resourceloaders.LoadRoles,
		Subs: []resourcekit.SubField{
			{
				Key:      "owner",
				Populate: populateOwnerOnRole,
			},
			{
				Key:         "owner.account",
				Target:      constants.ObjectTypeAccount,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractOwnerAccountIDFromRole,
				Populate:    populateOwnerAccountOnRole,
			},
			{
				Key:      "permissions",
				Populate: populatePermissionsOnRole,
			},
		},
	})
}

func populateOwnerOnRole(ctx context.Context, parent any, _ map[string]any) {
	r := parent.(*apiresource.Role)
	accountID, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeRole, r.ID, "owner_account_id")
	r.Owner = buildOwnerShell(accountID)
}

func extractOwnerAccountIDFromRole(ctx context.Context, parent any) []string {
	r := parent.(*apiresource.Role)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeRole, r.ID, "owner_account_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateOwnerAccountOnRole(ctx context.Context, parent any, loaded map[string]any) {
	r := parent.(*apiresource.Role)
	if r.Owner == nil {
		return
	}
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeRole, r.ID, "owner_account_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		r.Owner.Account = v.(*apiresource.Account)
	}
}

func populatePermissionsOnRole(ctx context.Context, parent any, _ map[string]any) {
	r := parent.(*apiresource.Role)
	perms, ok := resourcekit.GetLoadMeta(ctx).
		GetStrings(constants.ObjectTypeRole, r.ID, "permissions")
	if !ok {
		return
	}
	r.Permissions = &perms
}
