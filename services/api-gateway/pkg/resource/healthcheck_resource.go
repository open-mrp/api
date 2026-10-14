package apiresource

import (
	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
)

// A liveness report for the API.
type Healthcheck struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=healthcheck"`
	// Current operational status of the API.
	//
	// Always `healthy` on a successful response: no other value is ever reported, so treat the HTTP status code, not this field, as the real signal.
	Status string `json:"status" validate:"required"`
}

var exampleHealthcheck = &Healthcheck{
	Object: constants.ObjectTypeHealthcheck,
	Status: "healthy",
}

func (*Healthcheck) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(exampleHealthcheck)
}
