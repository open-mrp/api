package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to validate units by abbreviation.
type ValidateUnitsRequest struct {
	// Map of arbitrary keys to unit abbreviation values to validate.
	UnitMap map[string]string `json:"unit_map" validate:"required"`
}

var sampleValidateUnitsRequest = &ValidateUnitsRequest{
	UnitMap: map[string]string{"0": "kg"},
}

func (*ValidateUnitsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleValidateUnitsRequest)
}

// Validates unit abbreviations and returns matching units keyed by the original map keys.
type ValidateUnitsEndpoint struct{}

func (e *ValidateUnitsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ValidateUnitsRequest, *apiresource.ValidateUnitsResponse] {
	return (&apiendpoint.APIEndpoint[*ValidateUnitsRequest, *apiresource.ValidateUnitsResponse]{
		Title:             "Validate Units",
		Method:            http.MethodPut,
		Route:             "/v1/catalog/units/actions/validate",
		ContentType:       "application/json",
		Request:           &ValidateUnitsRequest{},
		Response:          &apiresource.ValidateUnitsResponse{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ValidateUnitsRequest) (*apiresource.ValidateUnitsResponse, *apierror.APIError) {
			return svc.(UnitSvc).ValidateUnits
		},
	}).WithDocSource(e)
}
