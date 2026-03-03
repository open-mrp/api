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

const createUnitEndpointDescription string = `This endpoint creates a new account-owned unit.`

// CreateUnitRequest is the request to create a new unit.
type CreateUnitRequest struct {
	// The display name of the unit (e.g. "Gram").
	Name string `json:"name" validate:"required"`
	// The short abbreviation for the unit (e.g. "g").
	Abbreviation string `json:"abbreviation" validate:"required"`
	// The unit dimension code (e.g. "mass", "quantity").
	Type constants.UnitType `json:"type" validate:"required"`
	// The conversion ratio numerator relative to the base unit, as a decimal string.
	RatioNumerator string `json:"ratio_numerator" validate:"required"`
	// The conversion ratio denominator relative to the base unit, as a decimal string.
	RatioDenominator string `json:"ratio_denominator" validate:"required"`
	// The conversion offset numerator, as a decimal string.
	OffsetNumerator string `json:"offset_numerator" validate:"required"`
	// The conversion offset denominator, as a decimal string.
	OffsetDenominator string `json:"offset_denominator" validate:"required"`
	// Whether this unit is the base unit for its dimension.
	IsBaseUnit bool `json:"is_base_unit"`
}

var sampleCreateUnitRequest = &CreateUnitRequest{
	Name:              "Gram",
	Abbreviation:      "g",
	Type:              constants.UnitTypeMass,
	RatioNumerator:    "1.000000000000000000000000000000",
	RatioDenominator:  "1.000000000000000000000000000000",
	OffsetNumerator:   "0.000000000000000000000000000000",
	OffsetDenominator: "1.000000000000000000000000000000",
	IsBaseUnit:        true,
}

func (*CreateUnitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateUnitRequest)
}

type CreateUnitEndpoint struct{}

func (e *CreateUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateUnitRequest, *apiresource.Unit] {
	return &apiendpoint.APIEndpoint[*CreateUnitRequest, *apiresource.Unit]{
		Title:             "Create Unit",
		Description:       createUnitEndpointDescription,
		Method:            http.MethodPost,
		Route:             "/v1/core/units",
		Request:           &CreateUnitRequest{},
		Response:          apiresource.SampleUnit,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateUnitRequest) (*apiresource.Unit, *apierror.APIError) {
			return svc.(UnitSvc).CreateUnit
		},
		Extras: apiendpoint.APIEndpointExtras{
			AllowUnknownJSONFields: false,
		},
	}
}
