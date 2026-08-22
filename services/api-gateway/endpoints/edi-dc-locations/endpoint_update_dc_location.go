package edidclocationep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to partially update a DC location.
type UpdateDCLocationRequest struct {
	// DC location ID.
	DCLocationID string `path:"id" validate:"required"`
	// ID of the customer account to reassign this DC location to.
	CustomerID field.Optional[string] `json:"customer_id,omitzero" validate:"omitempty"`
	// Free-form description identifying the distribution-center location, such as a warehouse name and bay (for example, `Warehouse B - Bay 1`).
	Location field.Optional[string] `json:"location,omitzero" validate:"omitempty,max=255"`
}

var sampleUpdateDCLocationLocation = "Warehouse B - Bay 1"
var sampleUpdateDCLocationRequest = &UpdateDCLocationRequest{
	Location: field.Some(sampleUpdateDCLocationLocation),
}

func (*UpdateDCLocationRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateDCLocationRequest)
}

// Partially updates a DC location.
//
// Omitted fields are left unchanged.
type UpdateDCLocationEndpoint struct{}

func (e *UpdateDCLocationEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateDCLocationRequest, *apiresource.DCLocation] {
	return (&apiendpoint.APIEndpoint[*UpdateDCLocationRequest, *apiresource.DCLocation]{
		Title:             "Update DC Location",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/operations/dc-locations/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainEdiRuns, Action: types.ActionUpdate},
		},
		ObjectType: constants.ObjectTypeDCLocation,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateDCLocationRequest) (*apiresource.DCLocation, *apierror.APIError) {
			return svc.(EDIDCLocationSvc).UpdateDCLocation
		},
	})
}
