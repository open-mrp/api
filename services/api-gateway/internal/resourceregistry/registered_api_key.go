package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeAPIKey,
		Load:       resourceloaders.LoadAPIKeys,
		Subs: []resourcekit.SubField{
			{
				Key:         "role",
				Target:      constants.ObjectTypeRole,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractRoleIDFromAPIKey,
				Populate:    populateRoleOnAPIKey,
			},
		},
	})

	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeCreatedAPIKey,
		Load: func(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
			return nil, nil
		},
		Subs: []resourcekit.SubField{
			{
				Key:         "role",
				Target:      constants.ObjectTypeRole,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractRoleIDFromCreatedAPIKey,
				Populate:    populateRoleOnCreatedAPIKey,
			},
		},
	})
}

func extractRoleIDFromAPIKey(ctx context.Context, parent any) []string {
	k := parent.(*apiresource.APIKey)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAPIKey, k.ID, "role_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateRoleOnAPIKey(ctx context.Context, parent any, loaded map[string]any) {
	k := parent.(*apiresource.APIKey)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAPIKey, k.ID, "role_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		k.Role = v.(*apiresource.Role)
	}
}

func extractRoleIDFromCreatedAPIKey(ctx context.Context, parent any) []string {
	c := parent.(*apiresource.CreatedAPIKey)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAPIKey, c.APIKeyInfo.ID, "role_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateRoleOnCreatedAPIKey(ctx context.Context, parent any, loaded map[string]any) {
	c := parent.(*apiresource.CreatedAPIKey)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAPIKey, c.APIKeyInfo.ID, "role_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		c.APIKeyInfo.Role = v.(*apiresource.Role)
	}
}
