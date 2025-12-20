package apiresource

import (
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
)

// Healthcheck contains information on the health of the application.
type Healthcheck struct {
	// Current operational status of the API service
	Status string `json:"status" validate:"required" example:"healthy"`
}

var exampleHealthcheck = &Healthcheck{
	Status: "healthy",
}

func (*Healthcheck) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(exampleHealthcheck)
}
