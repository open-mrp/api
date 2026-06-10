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
		ObjectType: constants.ObjectTypeAccountUser,
		Load:       resourceloaders.LoadAccountUsers,
		Subs: []resourcekit.SubField{
			{
				Key:         "user",
				Target:      constants.ObjectTypeUser,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractUserIDFromAccountUser,
				Populate:    populateUserOnAccountUser,
			},
			{
				Key:         "role",
				Target:      constants.ObjectTypeRole,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractRoleIDFromAccountUser,
				Populate:    populateRoleOnAccountUser,
			},
			{
				Key:         "department",
				Target:      constants.ObjectTypeDepartment,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractDepartmentIDFromAccountUser,
				Populate:    populateDepartmentOnAccountUser,
			},
		},
	})
}

func extractUserIDFromAccountUser(ctx context.Context, parent any) []string {
	au := parent.(*apiresource.AccountUser)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountUser, au.ID, "user_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateUserOnAccountUser(ctx context.Context, parent any, loaded map[string]any) {
	au := parent.(*apiresource.AccountUser)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountUser, au.ID, "user_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		au.User = v.(*apiresource.User)
	}
}

func extractRoleIDFromAccountUser(ctx context.Context, parent any) []string {
	au := parent.(*apiresource.AccountUser)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountUser, au.ID, "role_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateRoleOnAccountUser(ctx context.Context, parent any, loaded map[string]any) {
	au := parent.(*apiresource.AccountUser)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountUser, au.ID, "role_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		au.Role = v.(*apiresource.Role)
	}
}

func extractDepartmentIDFromAccountUser(ctx context.Context, parent any) []string {
	au := parent.(*apiresource.AccountUser)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountUser, au.ID, "department_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateDepartmentOnAccountUser(ctx context.Context, parent any, loaded map[string]any) {
	au := parent.(*apiresource.AccountUser)
	id, _ := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountUser, au.ID, "department_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		au.Department = v.(*apiresource.Department)
	}
}
