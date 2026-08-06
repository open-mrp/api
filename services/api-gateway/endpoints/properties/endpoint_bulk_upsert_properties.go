package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// carries one attribute under a bulk-upserted property
type UpsertPropertyAttributeInput struct {
	// The selectable value this attribute represents, such as `Red`.
	//
	// Must be unique across all attributes in the account, not just within the property. Leading and trailing whitespace is trimmed.
	Value string `json:"value" validate:"required,max=255"`
	// Swatch color used to display this attribute in the UI.
	//
	// When omitted, one of the nine named colors is assigned. Ignored for a value the property already defines.
	ColorCode field.Optional[constants.Color] `json:"color,omitzero"`
}

// carries one property in a bulk upsert
type UpsertPropertyInput struct {
	// Display name of the property, used to match existing properties within the account.
	Name string `json:"name" validate:"required,max=255"`
	// The selectable values to define under this property, in the order they should be arranged.
	//
	// Additive — values the property already defines are left as they stand, and none are removed. New values are appended after the existing ones.
	Attributes []UpsertPropertyAttributeInput `json:"attributes" default:"[]" validate:"max=1000,dive"`
}

// carries the properties to bulk upsert
type BulkUpsertPropertiesRequest struct {
	// Properties to create or update, matched by name (case-insensitive) within the account.
	Properties []UpsertPropertyInput `json:"properties" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertPropertiesRequest = &BulkUpsertPropertiesRequest{
	Properties: []UpsertPropertyInput{
		{
			Name: apiresource.SamplePropertyName,
			Attributes: []UpsertPropertyAttributeInput{
				{Value: apiresource.SampleAttributeValue, ColorCode: field.Some(constants.ColorRed)},
			},
		},
	},
}

func (*BulkUpsertPropertiesRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertPropertiesRequest)
}

// Creates or updates multiple properties and their attributes for the account, matched by
// name (case-insensitive), then writes asynchronously — 202 with a job to poll.
type BulkUpsertPropertiesEndpoint struct{}

func (e *BulkUpsertPropertiesEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertPropertiesRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertPropertiesRequest, *apiresource.Job]{
		Title:             "Bulk Upsert Properties",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/properties/actions/bulk-upsert",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertPropertiesRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(PropertySvc).BulkUpsertProperties
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
