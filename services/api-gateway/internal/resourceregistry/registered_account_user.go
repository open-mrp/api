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
		ObjectType: constants.ObjectTypeAccountUser,
		Load:       resourceloaders.LoadAccountUsers,
		Subs: []include.SubField{
			{
				Key:         "user",
				Target:      constants.ObjectTypeUser,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractUserIDFromAccountUser,
				Populate:    populateUserOnAccountUser,
			},
			{
				Key:         "role",
				Target:      constants.ObjectTypeRole,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractRoleIDFromAccountUser,
				Populate:    populateRoleOnAccountUser,
			},
			{
				Key:         "work_center",
				Target:      constants.ObjectTypeWorkCenter,
				Cardinality: include.CardinalityOnePtr,
				ExtractIDs:  extractWorkCenterIDFromAccountUser,
				Populate:    populateWorkCenterOnAccountUser,
			},
		},
	})
}

func extractUserIDFromAccountUser(ctx context.Context, parent any) []string {
	au := parent.(*apiresource.AccountUser)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountUser, au.ID, "user_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateUserOnAccountUser(ctx context.Context, parent any, loaded map[string]any) {
	au := parent.(*apiresource.AccountUser)
	id, _ := include.GetLoadMeta(ctx).
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
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountUser, au.ID, "role_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateRoleOnAccountUser(ctx context.Context, parent any, loaded map[string]any) {
	au := parent.(*apiresource.AccountUser)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountUser, au.ID, "role_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		au.Role = v.(*apiresource.Role)
	}
}

func extractWorkCenterIDFromAccountUser(ctx context.Context, parent any) []string {
	au := parent.(*apiresource.AccountUser)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountUser, au.ID, "department_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateWorkCenterOnAccountUser(ctx context.Context, parent any, loaded map[string]any) {
	au := parent.(*apiresource.AccountUser)
	id, _ := include.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAccountUser, au.ID, "department_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		au.WorkCenter = v.(*apiresource.WorkCenter)
	}
}
