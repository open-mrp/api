package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

const updateUnitEndpointDescription string = `This endpoint partially updates an account-owned unit.
Only provided fields are updated; absent fields retain their current values.
System units cannot be updated.`

// UpdateUnitRequest is the request to partially update a unit.
type UpdateUnitRequest struct {
	// The ID of the unit to update.
	UnitID string `path:"id"`
	// The display name of the unit.
	Name *string `json:"name,omitempty"`
	// The short abbreviation for the unit.
	Abbreviation *string `json:"abbreviation,omitempty"`
	// The conversion ratio numerator, as a decimal string.
	RatioNumerator *string `json:"ratio_numerator,omitempty"`
	// The conversion ratio denominator, as a decimal string.
	RatioDenominator *string `json:"ratio_denominator,omitempty"`
	// The conversion offset numerator, as a decimal string.
	OffsetNumerator *string `json:"offset_numerator,omitempty"`
	// The conversion offset denominator, as a decimal string.
	OffsetDenominator *string `json:"offset_denominator,omitempty"`
}

type UpdateUnitEndpoint struct{}

func (e *UpdateUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateUnitRequest, *apiresource.Unit] {
	return &apiendpoint.APIEndpoint[*UpdateUnitRequest, *apiresource.Unit]{
		Title:             "Update Unit",
		Description:       updateUnitEndpointDescription,
		Method:            http.MethodPatch,
		Route:             "/v1/core/units/{id}",
		ContentType:       "application/json",
		Request:           &UpdateUnitRequest{},
		Response:          apiresource.SampleUnit,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateUnitRequest) (*apiresource.Unit, *apierror.APIError) {
			return svc.(UnitSvc).UpdateUnit
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
