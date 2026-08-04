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
	// Abbreviations to look up, keyed by any identifier you choose.
	//
	// The keys are echoed back on the matching units, so a spreadsheet import can use row numbers or column names to trace each abbreviation back to its source.
	UnitMap map[string]string `json:"unit_map" validate:"required"`
}

var sampleValidateUnitsRequest = &ValidateUnitsRequest{
	UnitMap: map[string]string{"0": "kg"},
}

func (*ValidateUnitsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleValidateUnitsRequest)
}

// Resolves a batch of unit abbreviations to the units they refer to.
//
// Each abbreviation is matched case-insensitively against the account's units, including shared system units, and returned under the key it was sent with. Keys whose abbreviation matches no unit are omitted from the response, which is how invalid abbreviations are identified.
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
		Extras: apiendpoint.APIEndpointExtras{HideFromRequestLog: true},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ValidateUnitsRequest) (*apiresource.ValidateUnitsResponse, *apierror.APIError) {
			return svc.(UnitSvc).ValidateUnits
		},
	})
}
