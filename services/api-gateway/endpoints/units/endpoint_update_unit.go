package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateUnitRequest is the request to partially update a unit.
type UpdateUnitRequest struct {
	// The ID of the unit to update.
	UnitID string `path:"id" validate:"required"`
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

var sampleUpdateUnitRequest = &UpdateUnitRequest{
	Name:         new("Kilogram"),
	Abbreviation: new("kg"),
}

func (*UpdateUnitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateUnitRequest)
}

type UpdateUnitEndpoint struct{}

func (e *UpdateUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateUnitRequest, *apiresource.Unit] {
	return &apiendpoint.APIEndpoint[*UpdateUnitRequest, *apiresource.Unit]{
		Title:             "Update Unit",
		Description:       "Partially updates an account-owned unit; system units cannot be updated.",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/units/{id}",
		ContentType:       "application/json",
		Request:           &UpdateUnitRequest{},
		Response:          &apiresource.Unit{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateUnitRequest) (*apiresource.Unit, *apierror.APIError) {
			return svc.(UnitSvc).UpdateUnit
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnit,
			Fields:     []string{"owner"},
		}),
	}
}
