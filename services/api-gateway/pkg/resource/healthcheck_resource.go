package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
)

// Healthcheck contains information on the health of the application.
type Healthcheck struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=healthcheck"`
	// Current operational status of the API service.
	Status string `json:"status" validate:"required"`
}

var exampleHealthcheck = &Healthcheck{
	Object: constants.ObjectTypeHealthcheck,
	Status: "healthy",
}

func (*Healthcheck) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(exampleHealthcheck)
}
