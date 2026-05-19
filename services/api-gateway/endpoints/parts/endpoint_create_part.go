package partep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apirequest "github.com/augno/api/services/api-gateway/pkg/request"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/patch"
)

// Request to create a part.
type CreatePartRequest struct {
	// SKU.
	SKU string `json:"sku" validate:"required,max=255"`
	// Description.
	Description patch.Nullable[string] `json:"description,omitzero"`
	// Notes.
	Notes patch.Nullable[string] `json:"notes,omitzero"`
	// Category ID.
	CategoryID string `json:"category_id" validate:"required"`
	// Initial unit price. When set, numerator must be a currency unit and
	// denominator must not be.
	UnitPrice patch.Nullable[apirequest.RateInput] `json:"unit_price,omitzero"`
	// Initial unit cost. Same currency rule as unit_price.
	UnitCost patch.Nullable[apirequest.RateInput] `json:"unit_cost,omitzero"`
	// Initial burn rate (waste / scrap). No currency requirement.
	BurnRate patch.Nullable[apirequest.RateInput] `json:"burn_rate,omitzero"`
	// Attribute IDs to connect to the part at creation time.
	AttributeIDs []string `json:"attribute_ids,omitempty"`
}

var sampleCreatePartDescription = "Deep groove ball bearing, 20x47x14mm"
var sampleCreatePartRequest = &CreatePartRequest{
	SKU:         apiresource.SamplePartSKU,
	Description: patch.PtrNullable(&sampleCreatePartDescription),
	CategoryID:  apiresource.SampleItemCategoryID,
}

func (*CreatePartRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreatePartRequest)
}

// Creates a part with the specified SKU and category.
type CreatePartEndpoint struct{}

func (e *CreatePartEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreatePartRequest, *apiresource.Part] {
	return (&apiendpoint.APIEndpoint[*CreatePartRequest, *apiresource.Part]{
		Title:             "Create Part",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/parts",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreatePartRequest) (*apiresource.Part, *apierror.APIError) {
			return svc.(PartSvc).CreatePart
		},
		LocationFunc: func(resp *apiresource.Part) string {
			return "/v1/catalog/parts/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePart,
			Fields:     []string{"item", "item.category", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
