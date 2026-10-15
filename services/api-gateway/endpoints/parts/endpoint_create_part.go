package partep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
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
	// `per_unit_id` must not be a currency unit (e.g. `$5` per `ea`). When omitted, the price is initialized to a zero price per the category's base unit.
	UnitPrice field.Optional[apirequest.UnitPriceInput] `json:"unit_price,omitzero"`
	// Initial cost per unit.
	//
	// Follows the same `per_unit_id` rule as `unit_price`. When omitted, the cost is initialized to a zero price per the category's base unit.
	UnitCost field.Optional[apirequest.UnitPriceInput] `json:"unit_cost,omitzero" sensitive:"cost"`
	// IDs of existing attributes to link to the part at creation time.
	//
	// Each attribute's property must be one the part's category carries; an attribute from any other property fails the whole request.
	AttributeIDs []string `json:"attribute_ids,omitzero"`
}

var sampleCreatePartDescription = "Deep groove ball bearing, 20x47x14mm"
var sampleCreatePartNotes = "OEM-equivalent; verify shielding type before substitution."
var sampleCreatePartRequest = &CreatePartRequest{
	SKU:          apiresource.SamplePartSKU,
	Description:  field.SomePtr(&sampleCreatePartDescription),
	Notes:        field.Some(sampleCreatePartNotes),
	CategoryID:   apiresource.SampleItemCategoryID,
	UnitPrice:    field.Some(apirequest.UnitPriceInput{Amount: "14.99", Currency: "usd", PerUnitID: apiresource.SampleUnitID}),
	UnitCost:     field.Some(apirequest.UnitPriceInput{Amount: "9.40", Currency: "usd", PerUnitID: apiresource.SampleUnitID}),
	AttributeIDs: []string{apiresource.SampleAttributeID},
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
		Title:             "Create Part",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/parts",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainParts, Action: types.ActionCreate}}, Counterparty: authz.Counterparties(types.ActionUpdate)},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreatePartRequest) (*apiresource.Part, *apierror.APIError) {
			return svc.(PartSvc).CreatePart
		},
		LocationFunc: func(resp *apiresource.Part) string {
			return "/v1/catalog/parts/" + resp.ID
		},
		ObjectType: constants.ObjectTypePart,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePart,
			Fields:     []string{"item", "item.category", "item.attributes"},
		}),
	})
}
