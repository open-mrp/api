package apiresource

import apiexample "github.com/augno/api/services/api-gateway/pkg/example"

var SampleLightRole = &LightRole{
	ID:   SampleRoleID,
	Name: SampleRoleName,
}

// LightRole represents a minimal role reference
type LightRole struct {
	// The unique identifier for the role
	ID string `json:"id" validate:"required"`
	// The display name of the role
	Name string `json:"name" validate:"required"`
}

func (*LightRole) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleLightRole)
}
