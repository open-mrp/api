package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	// Sandbox exposes a direct `owner_account` expansion (an Account) — no
	// intermediate Owner shell. The SubField reuses the already-registered
	// Account loader; the FK is stashed in LoadMeta by LoadSandboxes.
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeSandbox,
		Load:       resourceloaders.LoadSandboxes,
		Subs: []resourcekit.SubField{
			{
				Key:         "owner_account",
				Target:      constants.ObjectTypeAccount,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractOwnerAccountIDFromSandbox,
				Populate:    populateOwnerAccountOnSandbox,
			},
		},
	})
}

func extractOwnerAccountIDFromSandbox(ctx context.Context, parent any) []string {
	s := parent.(*apiresource.Sandbox)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeSandbox, s.ID, "owner_account_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateOwnerAccountOnSandbox(ctx context.Context, parent any, loaded map[string]any) {
	s := parent.(*apiresource.Sandbox)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeSandbox, s.ID, "owner_account_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		s.OwnerAccount = v.(*apiresource.Account)
	}
}
