package propertyep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// carries one attribute under a bulk-upserted property
type UpsertPropertyAttributeInput struct {
	// The value, such as `Large`. Matched case-insensitively against the property's existing attributes; leading and trailing whitespace is trimmed.
	Value string `json:"value" validate:"required,max=255"`
	// Swatch color for a new attribute. Defaults to `default`. Ignored for a value the property already has.
	ColorCode field.Optional[constants.Color] `json:"color,omitzero"`
}

// carries one property in a bulk upsert
type UpsertPropertyInput struct {
	// Display name, such as `Size`. Matches an existing property by name (case-insensitive).
	Name string `json:"name" validate:"required,max=255"`
	// Attributes to add, by value, appended in order. Existing attributes are never changed or removed.
	Attributes []UpsertPropertyAttributeInput `json:"attributes" default:"[]" validate:"max=1000,dive"`
}

// carries the properties to bulk upsert
type BulkUpsertPropertiesRequest struct {
	// Properties to create or update.
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

// Creates or updates properties and their attributes in bulk.
//
// Rows are matched to existing properties by `name` (case-insensitive). Attributes are added by value and appended in order. Existing attributes are never changed or removed. The work runs in the background; the response is an async job to poll.
type BulkUpsertPropertiesEndpoint struct{}

func (e *BulkUpsertPropertiesEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertPropertiesRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertPropertiesRequest, *apiresource.AsyncJob]{
		Title:             "Bulk Upsert Properties",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/properties/actions/bulk-upsert",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainProperties, Action: types.ActionCreate}, {Domain: types.PermissionDomainProperties, Action: types.ActionUpdate}}, RequireAll: true},
		ObjectType:        constants.ObjectTypeAsyncJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertPropertiesRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(PropertySvc).BulkUpsertProperties
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
