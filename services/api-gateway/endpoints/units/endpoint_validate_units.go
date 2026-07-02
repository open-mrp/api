package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	apierror "github.com/augno/api/shared/errors"
)

// Request to validate units by abbreviation.
type ValidateUnitsRequest struct {
	// Map of arbitrary keys to unit abbreviations to validate.
	//
	// Abbreviations are matched case-insensitively against the account's units.
	UnitMap map[string]string `json:"unit_map" validate:"required"`
}

var sampleValidateUnitsRequest = &ValidateUnitsRequest{
	UnitMap: map[string]string{"0": "kg"},
}

func (*ValidateUnitsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleValidateUnitsRequest)
}

// Looks up units by abbreviation and returns the matches keyed by the original map keys; keys with no matching unit are omitted from the response.
type ValidateUnitsEndpoint struct{}

func (e *ValidateUnitsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ValidateUnitsRequest, *apiresource.ValidateUnitsResponse] {
	return (&apiendpoint.APIEndpoint[*ValidateUnitsRequest, *apiresource.ValidateUnitsResponse]{
		Title:             "Validate Units",
		Method:            http.MethodPut,
		Route:             "/v1/catalog/units/actions/validate",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainUnits, Action: types.ActionRead},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ValidateUnitsRequest) (*apiresource.ValidateUnitsResponse, *apierror.APIError) {
			return svc.(UnitSvc).ValidateUnits
		},
	})
}
