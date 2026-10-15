package locationep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
)

// UpsertLocationInput is the input for a single location in a bulk upsert operation.
type UpsertLocationInput struct {
	// Display name of the location, used to match existing locations (case-insensitive).
	Name string `json:"name" validate:"required,max=255"`
	// What kind of place this is: `site`, `building`, `section`, `aisle`, `rack`, `shelf` or `bin`. A site is always top-level.
	TypeCode constants.LocationTypeCode `json:"type" validate:"required"`
	// `active` or `archived`. Omitted, a new location is active and an existing one keeps its status.
	Status *constants.ArchiveStatus `json:"status,omitempty"`
	// Parent location, referenced by `id` or `name`, or by name for a location in the same
	// batch. Omitted, a new location is top-level and an existing one keeps its parent. A site takes no parent.
	Parent *apirequest.ObjectIdentifier `json:"parent,omitempty"`
	// Child locations to re-parent under this one, referenced by `id` or `name`, or by name
	// for a location in the same batch. Redundant with `parent` on each child.
	Children []apirequest.ObjectIdentifier `json:"children,omitempty"`
}

// BulkUpsertLocationsRequest is the request to bulk upsert locations.
type BulkUpsertLocationsRequest struct {
	// Locations to create or update, matched by name within the account.
	Locations []UpsertLocationInput `json:"locations" validate:"required,min=1,max=1000,dive"`
}

var sampleBulkUpsertLocationsRequest = &BulkUpsertLocationsRequest{
	Locations: []UpsertLocationInput{
		{
			Name:     apiresource.SampleLocationName,
			TypeCode: apiresource.SampleLocationTypeCode,
		},
	},
}

func (*BulkUpsertLocationsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBulkUpsertLocationsRequest)
}

// Creates or updates many locations at once.
//
// Rows are matched to your existing locations by name (case-insensitive). The writes run in the background: the response is a 202 with an async job to poll for each row's result.
type BulkUpsertLocationsEndpoint struct{}

func (e *BulkUpsertLocationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertLocationsRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertLocationsRequest, *apiresource.AsyncJob]{
		Title:             "Bulk Upsert Locations",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/locations/actions/bulk-upsert",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainLocations, Action: types.ActionCreate}, {Domain: types.PermissionDomainLocations, Action: types.ActionUpdate}}, RequireAll: true},
		ObjectType:        constants.ObjectTypeAsyncJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeAsyncJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertLocationsRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(LocationSvc).BulkUpsertLocations
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}
