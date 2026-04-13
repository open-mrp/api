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

// Request to partially update a product line.
type UpdateProductLineRequest struct {
	// Product line ID.
	ProductLineID string `path:"id" validate:"required"`
	// Display name.
	Name *string `json:"name,omitempty" nullable:"false" validate:"omitempty,max=255"`
	// Commission policy.
	CommissionPolicy *constants.CommissionPolicy `json:"commission_policy,omitempty" nullable:"false"`
	// Freight policy.
	FreightPolicy *constants.FreightPolicy `json:"freight_policy,omitempty" nullable:"false"`
	// Unit group ID.
	UnitGroupID *string `json:"unit_group_id,omitempty" nullable:"false" validate:"omitempty,max=191"`
}

var sampleUpdateProductLineName = "Updated Product Line"

var sampleUpdateProductLineRequest = &UpdateProductLineRequest{
	Name: &sampleUpdateProductLineName,
}

func (*UpdateProductLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductLineRequest)
}

type UpdateProductLineEndpoint struct{}

func (e *UpdateProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductLineRequest, *apiresource.ProductLine] {
	return &apiendpoint.APIEndpoint[*UpdateProductLineRequest, *apiresource.ProductLine]{
		Title:             "Update Product Line",
		Description:       "Partially updates an account-owned product line. Default system product lines cannot be updated.",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/product-lines/{id}",
		ContentType:       "application/json",
		Request:           &UpdateProductLineRequest{},
		Response:          &apiresource.ProductLine{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
			return svc.(ProductLineSvc).UpdateProductLine
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductLine,
			Fields:     []string{"owner", "owner.account", "unit_group"},
		}),
	}
}
