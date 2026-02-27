package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

var SampleLightRole = &LightRole{
	ID:           SampleRoleID,
	Name:         SampleRoleName,
	ObjectType:   constants.ObjectTypeRole,
	RoleTypeCode: new(constants.RoleTypeCodeAdmin),
}

// LightRole represents a minimal role reference.
type LightRole struct {
	// The unique identifier for the role.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	ObjectType constants.ObjectType `json:"object_type" validate:"required,enum=role"`
	// The display name of the role.
	Name string `json:"name" validate:"required"`
	// The role type code.
	RoleTypeCode *constants.RoleTypeCode `json:"role_type_code"`
}

func (*LightRole) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleLightRole)
}
