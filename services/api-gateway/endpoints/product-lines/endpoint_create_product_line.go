package productlineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// CreateProductLineRequest is the request to create a new product line.
type CreateProductLineRequest struct {
	// The display name of the product line.
	Name string `json:"name" validate:"required,max=255"`
	// The ID of the unit group to associate with this product line.
	UnitGroupID string `json:"unit_group_id" validate:"required,max=191"`
	// The commission policy for this product line.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// The freight policy for this product line.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
}

var sampleCreateProductLineRequest = &CreateProductLineRequest{
	Name:             apiresource.SampleProductLineName,
	UnitGroupID:      apiresource.SampleUnitGroupID,
	CommissionPolicy: constants.CommissionPolicyExempt,
	FreightPolicy:    constants.FreightPolicyBilled,
}

func (*CreateProductLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateProductLineRequest)
}

type CreateProductLineEndpoint struct{}

func (e *CreateProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductLineRequest, *apiresource.ProductLine] {
	return &apiendpoint.APIEndpoint[*CreateProductLineRequest, *apiresource.ProductLine]{
		Title:             "Create Product Line",
		Description:       "Creates a new account-owned product line.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/catalog/product-lines",
		Request:           &CreateProductLineRequest{},
		Response:          &apiresource.ProductLine{},
		SuccessStatusCode: http.StatusCreated,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
			return svc.(ProductLineSvc).CreateProductLine
		},
		LocationFunc: func(resp *apiresource.ProductLine) string {
			return "/v1/catalog/product-lines/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductLine,
			Fields:     []string{"owner", "owner.account", "unit_group"},
		}),
	}
}
