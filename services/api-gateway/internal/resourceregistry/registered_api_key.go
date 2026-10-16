package resourceregistry

import (
	"context"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeAPIKey,
		Load:       resourceloaders.LoadAPIKeys,
		Subs: []include.SubField{
			{
				Key:         "role",
				Target:      constants.ObjectTypeRole,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractRoleIDFromAPIKey,
				Populate:    populateRoleOnAPIKey,
			},
		},
	})

	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeCreatedAPIKey,
		Load: func(_ context.Context, _ []string) (map[string]any, *apierror.APIError) {
			return nil, nil
		},
		Subs: []include.SubField{
			{
				Key:         "role",
				Target:      constants.ObjectTypeRole,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractRoleIDFromCreatedAPIKey,
				Populate:    populateRoleOnCreatedAPIKey,
			},
		},
	})
}

func extractRoleIDFromAPIKey(ctx context.Context, parent any) []string {
	k := parent.(*apiresource.APIKey)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAPIKey, k.ID, "role_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateRoleOnAPIKey(ctx context.Context, parent any, loaded map[string]any) {
	k := parent.(*apiresource.APIKey)
	id, _ := include.GetLoadMeta(ctx).
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
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAPIKey, c.APIKey.ID, "role_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateRoleOnCreatedAPIKey(ctx context.Context, parent any, loaded map[string]any) {
	c := parent.(*apiresource.CreatedAPIKey)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAPIKey, c.APIKey.ID, "role_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		c.APIKey.Role = v.(*apiresource.Role)
	}
}
