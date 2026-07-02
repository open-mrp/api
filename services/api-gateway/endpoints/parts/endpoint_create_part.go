package partep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to create a part.
type CreatePartRequest struct {
	// Stock keeping unit code for the part.
	//
	// Must be unique within the account; creating a part with a SKU already used by another item fails with a conflict error.
	SKU string `json:"sku" validate:"required,max=255"`
	// Free-form description of the part.
	Description field.Optional[string] `json:"description,omitzero"`
	// Free-form notes about the part.
	Notes field.Optional[string] `json:"notes,omitzero"`
	// ID of the item category to place the part in.
	//
	// The category's unit group determines the base unit used for the part's rates (`unit_value`, `unit_cost`, `burn_rate`).
	CategoryID string `json:"category_id" validate:"required"`
	// Initial selling price per unit.
	//
	// `numerator_unit_id` must reference a currency unit and `denominator_unit_id` must reference a non-currency unit (e.g. `$5` per `ea`). When omitted, the price is initialized to a zero rate in the category's base unit.
	UnitPrice field.Optional[apirequest.RateInput] `json:"unit_price,omitzero"`
	// Initial cost per unit.
	//
	// Follows the same unit rule as `unit_price`: currency numerator, non-currency denominator. When omitted, the cost is initialized to a zero rate in the category's base unit.
	UnitCost field.Optional[apirequest.RateInput] `json:"unit_cost,omitzero"`
	// IDs of existing attributes to link to the part at creation time.
	AttributeIDs []string `json:"attribute_ids,omitzero"`
}

var sampleCreatePartDescription = "Deep groove ball bearing, 20x47x14mm"
var sampleCreatePartRequest = &CreatePartRequest{
	SKU:         apiresource.SamplePartSKU,
	Description: field.SomePtr(&sampleCreatePartDescription),
	CategoryID:  apiresource.SampleItemCategoryID,
}

func (*CreatePartRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreatePartRequest)
}

// Creates a part with the specified SKU and category.
//
// Inventory tracking for the new part starts at a zero on-hand quantity in the category's base unit.
type CreatePartEndpoint struct{}

func (e *CreatePartEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreatePartRequest, *apiresource.Part] {
	return (&apiendpoint.APIEndpoint[*CreatePartRequest, *apiresource.Part]{
		Title:               "Create Part",
		Method:              http.MethodPost,
		Route:               "/v1/catalog/parts",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainParts, Action: types.ActionCreate}, {Domain: types.PermissionDomainCustomers, Action: types.ActionUpdate}, {Domain: types.PermissionDomainSuppliers, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreatePartRequest) (*apiresource.Part, *apierror.APIError) {
			return svc.(PartSvc).CreatePart
		},
		LocationFunc: func(resp *apiresource.Part) string {
			return "/v1/catalog/parts/" + resp.ID
		},
		ObjectType: constants.ObjectTypePart,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePart,
			Fields:     []string{"item", "item.category", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
