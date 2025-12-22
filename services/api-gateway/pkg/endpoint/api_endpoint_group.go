package apiendpoint

import (
	"github.com/augno/api/shared/contracts"
)

/*
APIEndpointGroup groups related APIEndpoints, often revolving around a specific resource.
This will be used to generate the OpenAPI spec. Consequently, consider this public data.
*/
type APIEndpointGroup struct {
	Title        string                   `json:"title" yaml:"title"`
	Description  string                   `json:"description" yaml:"description"`
	ResourceType contracts.DocumentedType `json:"-" yaml:"-"`
	Endpoints    []APIEndpointer          `json:"endpoints" yaml:"endpoints"`
}
