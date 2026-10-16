package manufacturingmethodep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/field"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
)

// Request to update a material.
type UpdateMethodMaterialRequest struct {
	// Manufacturing method ID.
	ManufacturingMethodID string `path:"manufacturing_method_id" validate:"required"`
	// Material ID.
	ID string `path:"id" validate:"required"`
	// ID of the item consumed.
	ItemID field.Optional[string] `json:"item_id,omitzero"`
	// ID of the operation that consumes it.
	OperationID field.Optional[string] `json:"operation_id,omitzero"`
	// How much of the item is consumed per the method's output quantity.
	Quantity field.Optional[apirequest.QuantityInput] `json:"quantity,omitzero"`
	// The expected loss on top of `quantity`.
	ScrapQuantity field.Optional[apirequest.QuantityInput] `json:"scrap_quantity,omitzero"`
	// Where the material comes from: `make`, `stock` or `buy`.
	Supply field.Optional[constants.Supply] `json:"supply,omitzero"`
	// Instructions for handling the material; `null` removes them. Maximum 10,000 characters.
	WorkInstructions field.Clearable[string] `json:"work_instructions,omitzero" validate:"omitempty,max=10000"`
}

var sampleUpdateMethodMaterialRequest = &UpdateMethodMaterialRequest{
	Quantity: field.Some(apirequest.QuantityInput{Value: "1.1", UnitID: apiresource.SampleUnitID}),
}

func (*UpdateMethodMaterialRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateMethodMaterialRequest)
}

// Updates a material of a draft manufacturing method.
//
// Only the fields you send change. Only a `draft` method can change (409 `invalid_state_transition`).
type UpdateMethodMaterialEndpoint struct{}

func (e *UpdateMethodMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateMethodMaterialRequest, *apiresource.MethodMaterial] {
	return &apiendpoint.APIEndpoint[*UpdateMethodMaterialRequest, *apiresource.MethodMaterial]{
		Title:             "Update Method Material",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             materialsRoute + "/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeMethodMaterial,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateMethodMaterialRequest) (*apiresource.MethodMaterial, *apierror.APIError) {
			return svc.(ManufacturingMethodSvc).UpdateMethodMaterial
		},
		IncludeConfig: materialIncludes(),
	}
}
