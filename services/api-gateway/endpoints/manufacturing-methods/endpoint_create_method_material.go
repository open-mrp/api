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

// Request to add a material to a draft manufacturing method.
type CreateMethodMaterialRequest struct {
	// Manufacturing method ID.
	ManufacturingMethodID string `path:"manufacturing_method_id" validate:"required"`
	// ID of the item consumed. Can't be the item the method makes.
	ItemID string `json:"item_id" validate:"required"`
	// ID of the operation that consumes it. Defaults to the method's first operation.
	OperationID field.Optional[string] `json:"operation_id,omitzero"`
	// How much of the item is consumed per the method's output quantity.
	Quantity apirequest.QuantityInput `json:"quantity" validate:"required"`
	// The expected loss on top of `quantity`. Defaults to zero.
	ScrapQuantity field.Optional[apirequest.QuantityInput] `json:"scrap_quantity,omitzero"`
	// Where the material comes from: `make`, `stock` or `buy`. Defaults to `make` when the item has an active method, else `stock`.
	Supply field.Optional[constants.Supply] `json:"supply,omitzero"`
	// Instructions for handling the material in the operation. Maximum 10,000 characters.
	WorkInstructions field.Optional[string] `json:"work_instructions,omitzero" validate:"omitempty,max=10000"`
}

var sampleCreateMethodMaterialRequest = &CreateMethodMaterialRequest{
	ItemID:      apiresource.SampleItemID,
	OperationID: field.Some(apiresource.SampleMethodOperationID),
	Quantity:    apirequest.QuantityInput{Value: "1.05", UnitID: apiresource.SampleUnitID},
	Supply:      field.Some(constants.SupplyStock),
}

func (*CreateMethodMaterialRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateMethodMaterialRequest)
}

// Adds a material to a draft manufacturing method.
//
// Only a `draft` method can change (409 `invalid_state_transition`). The method needs an operation first. A method has at most 500 materials.
type CreateMethodMaterialEndpoint struct{}

func (e *CreateMethodMaterialEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateMethodMaterialRequest, *apiresource.MethodMaterial] {
	return &apiendpoint.APIEndpoint[*CreateMethodMaterialRequest, *apiresource.MethodMaterial]{
		Title:             "Create Method Material",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             materialsRoute,
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeMethodMaterial,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateMethodMaterialRequest) (*apiresource.MethodMaterial, *apierror.APIError) {
			return svc.(ManufacturingMethodSvc).CreateMethodMaterial
		},
		IncludeConfig: materialIncludes(),
	}
}
