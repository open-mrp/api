package apiendpoint

import (
	"github.com/open-mrp/api/shared/contracts"
)

/*
APIEndpointGroup groups related APIEndpoints, often revolving around a specific resource.
This will be used to generate the OpenAPI spec. Consequently, consider this public data.
*/
type APIEndpointGroup struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	// SDKResourcePath overrides the generated Stainless resource path for
	// endpoints in this group when route inference is insufficient.
	SDKResourcePath []string                 `json:"sdk_resource_path,omitempty" yaml:"sdk_resource_path,omitempty"`
	ResourceType    contracts.DocumentedType `json:"-" yaml:"-"`
	Endpoints       []APIEndpointer          `json:"endpoints" yaml:"endpoints"`
}
