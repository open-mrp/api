package unitep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
)

// Request to create a unit.
type CreateUnitRequest struct {
	// Display name, such as `Kilogram`.
	//
	// Unique (case-insensitive) among your account's units and the system units.
	Name string `json:"name" validate:"required,max=255"`
	// Short label, such as `kg`.
	//
	// Unique (case-insensitive) among your account's units and the system units.
	Abbreviation string `json:"abbreviation" validate:"required,max=32"`
	// What the unit measures.
	//
	// Values convert only between units of the same dimension. `quantity` is for countable things, such as each, pair or case. Fixed after creation.
	Dimension constants.UnitDimension `json:"dimension" validate:"required"`
	// The ratio that converts a value in this unit to its dimension's base unit.
	//
	// A value converts as `value × ratio_numerator ÷ ratio_denominator + offset_numerator ÷ offset_denominator`. A kilogram, with grams as the base, is `1000` / `1`.
	RatioNumerator string `json:"ratio_numerator" validate:"required,decimal" format:"decimal"`
	// The ratio that converts a value in this unit to its dimension's base unit.
	//
	// The denominator can't be zero.
	RatioDenominator string `json:"ratio_denominator" validate:"required,nonzero_decimal" format:"decimal"`
	// An offset added after the ratio, for scales with different zero points, such as temperature.
	//
	// Send `0` for units with no offset.
	OffsetNumerator string `json:"offset_numerator" validate:"required,decimal" format:"decimal"`
	// An offset added after the ratio, for scales with different zero points, such as temperature.
	//
	// The denominator can't be zero; send `1` for units with no offset.
	OffsetDenominator string `json:"offset_denominator" validate:"required,nonzero_decimal" format:"decimal"`
}

var sampleCreateUnitRequest = &CreateUnitRequest{
	Name:              "Gram",
	Abbreviation:      "g",
	Dimension:         constants.UnitDimensionMass,
	RatioNumerator:    "1",
	RatioDenominator:  "1",
	OffsetNumerator:   "0",
	OffsetDenominator: "1",
}

func (*CreateUnitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateUnitRequest)
}

// Creates a unit.
//
// The new unit belongs to your account. Its conversion ratio is relative to the base unit of its dimension.
type CreateUnitEndpoint struct{}

func (e *CreateUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateUnitRequest, *apiresource.Unit] {
	return (&apiendpoint.APIEndpoint[*CreateUnitRequest, *apiresource.Unit]{
		Title:             "Create Unit",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/units",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainUnits, Action: types.ActionCreate}}},
		ObjectType:        constants.ObjectTypeUnit,
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
