package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a unit.
type CreateUnitRequest struct {
	// Display name of the unit (e.g. "Gram").
	//
	// Must be unique within the account.
	Name string `json:"name" validate:"required,max=255"`
	// Short abbreviation for the unit (e.g. "g").
	//
	// Must be unique within the account.
	Abbreviation string `json:"abbreviation" validate:"required"`
	// Unit dimension.
	//
	// Units can only be converted to other units of the same dimension.
	Type constants.UnitType `json:"type" validate:"required"`
	// Conversion ratio numerator relative to the base unit.
	RatioNumerator string `json:"ratio_numerator" validate:"required,decimal" format:"decimal"`
	// Conversion ratio denominator relative to the base unit.
	//
	// Must not be zero.
	RatioDenominator string `json:"ratio_denominator" validate:"required,nonzero_decimal" format:"decimal"`
	// Conversion offset numerator, used for temperature-like conversions.
	OffsetNumerator string `json:"offset_numerator" validate:"required,decimal" format:"decimal"`
	// Conversion offset denominator.
	//
	// Must not be zero.
	OffsetDenominator string `json:"offset_denominator" validate:"required,nonzero_decimal" format:"decimal"`
}

var sampleCreateUnitRequest = &CreateUnitRequest{
	Name:              "Gram",
	Abbreviation:      "g",
	Type:              constants.UnitTypeMass,
	RatioNumerator:    "1",
	RatioDenominator:  "1",
	OffsetNumerator:   "0",
	OffsetDenominator: "1",
}

func (*CreateUnitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateUnitRequest)
}

// Creates an account-owned unit.
type CreateUnitEndpoint struct{}

func (e *CreateUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateUnitRequest, *apiresource.Unit] {
	return (&apiendpoint.APIEndpoint[*CreateUnitRequest, *apiresource.Unit]{
		Title:               "Create Unit",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/catalog/units",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainUnits, Action: types.ActionCreate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeUnit,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateUnitRequest) (*apiresource.Unit, *apierror.APIError) {
			return svc.(UnitSvc).CreateUnit
		},
		LocationFunc: func(resp *apiresource.Unit) string {
			return "/v1/catalog/units/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeUnit,
			Fields:     []string{"owner", "owner.account"},
		}),
	})
}
