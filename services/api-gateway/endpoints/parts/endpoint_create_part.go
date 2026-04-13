package partep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a part.
type CreatePartRequest struct {
	// SKU.
	SKU string `json:"sku" validate:"required,max=255"`
	// Description.
	Description *string `json:"description"`
	// Category ID.
	CategoryID string `json:"category_id" validate:"required,max=191"`
}

var sampleCreatePartDescription = "Deep groove ball bearing, 20x47x14mm"
var sampleCreatePartRequest = &CreatePartRequest{
	SKU:         apiresource.SamplePartSKU,
	Description: &sampleCreatePartDescription,
	CategoryID:  apiresource.SampleItemCategoryID,
}

func (*CreatePartRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreatePartRequest)
}

type CreatePartEndpoint struct{}

func (e *CreatePartEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreatePartRequest, *apiresource.Part] {
	return &apiendpoint.APIEndpoint[*CreatePartRequest, *apiresource.Part]{
		Title:             "Create Part",
		Description:       "Creates a part with the specified SKU and category.",
		Method:            http.MethodPost,
		Route:             "/v1/operations/parts",
		ContentType:       "application/json",
		Request:           &CreatePartRequest{},
		Response:          &apiresource.Part{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreatePartRequest) (*apiresource.Part, *apierror.APIError) {
			return svc.(PartSvc).CreatePart
		},
		LocationFunc: func(resp *apiresource.Part) string {
			return "/v1/operations/parts/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePart,
			Fields:     []string{"category", "unit_value", "unit_cost", "burn_rate"},
		}),
	}
}
