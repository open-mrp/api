package unitep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
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
	// The dimension this unit measures, such as mass, volume, or currency.
	//
	// Units can only be converted to other units of the same dimension, and the dimension cannot be changed after the unit is created.
	Type constants.UnitType `json:"type" validate:"required"`
	// Numerator of the ratio that converts a quantity in this unit into the dimension's base unit.
	//
	// A quantity is converted with `value × (ratio_numerator / ratio_denominator) + (offset_numerator / offset_denominator)`, so a kilogram in a gram-based dimension has a numerator of `1000` and a denominator of `1`.
	RatioNumerator string `json:"ratio_numerator" validate:"required,decimal" format:"decimal"`
	// Denominator of the ratio that converts a quantity in this unit into the dimension's base unit.
	//
	// Must not be zero.
	RatioDenominator string `json:"ratio_denominator" validate:"required,nonzero_decimal" format:"decimal"`
	// Numerator of the conversion offset, applied after the ratio for scales that do not share a zero point, such as temperature.
	//
	// Send `0` for units that convert by ratio alone.
	OffsetNumerator string `json:"offset_numerator" validate:"required,decimal" format:"decimal"`
	// Denominator of the conversion offset.
	//
	// Must not be zero, so send `1` when the unit has no offset.
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

// Creates a unit of measurement owned by your account, in addition to the system units the platform already provides.
//
// The name and abbreviation must each be unique within the account. A unit created here is never a base unit, so its conversion ratio is interpreted relative to the base unit of the chosen dimension.
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
