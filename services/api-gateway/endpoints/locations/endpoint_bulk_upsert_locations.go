package locationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// UpsertLocationInput is the input for a single location in a bulk upsert operation.
type UpsertLocationInput struct {
	// Display name of the location, used to match existing locations.
	Name string `json:"name" validate:"required,max=255"`
	// Location type code.
	TypeCode constants.LocationTypeCode `json:"type" validate:"required"`
	// Parent location, referenced by `id` or `name`, or by name for a location in the same
	// batch. Omitted, a new location is top-level and an existing one keeps its parent.
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

// Creates or updates multiple locations for the account, matched by name
// (case-insensitive), then writes asynchronously — 202 with a job to poll.
type BulkUpsertLocationsEndpoint struct{}

func (e *BulkUpsertLocationsEndpoint) Materialize() *apiendpoint.APIEndpoint[*BulkUpsertLocationsRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*BulkUpsertLocationsRequest, *apiresource.Job]{
		Title:             "Bulk Upsert Locations",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/operations/locations/actions/bulk-upsert",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *BulkUpsertLocationsRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(LocationSvc).BulkUpsertLocations
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
