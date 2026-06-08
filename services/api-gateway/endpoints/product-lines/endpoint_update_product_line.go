package productlineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to partially update a product line.
type UpdateProductLineRequest struct {
	// Product line ID.
	ProductLineID string `path:"id" validate:"required"`
	// Display name.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Commission policy of products in this product line.
	CommissionPolicy field.Optional[constants.CommissionPolicy] `json:"commission_policy,omitzero"`
	// Freight policy for all items in this product line.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero"`
	// Unit group ID associated with this product line. This unit group dictates the units that products in this product line may be purchased in.
	UnitGroupID field.Optional[string] `json:"unit_group_id,omitzero" validate:"omitempty"`
}

var sampleUpdateProductLineName = "Updated Product Line"

var sampleUpdateProductLineRequest = &UpdateProductLineRequest{
	Name: field.Some(sampleUpdateProductLineName),
}

func (*UpdateProductLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductLineRequest)
}

// Partially updates an account-owned product line. Default system product lines cannot be updated.
type UpdateProductLineEndpoint struct{}

func (e *UpdateProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductLineRequest, *apiresource.ProductLine] {
	return (&apiendpoint.APIEndpoint[*UpdateProductLineRequest, *apiresource.ProductLine]{
		Title:             "Update Product Line",
		Method:            http.MethodPatch,
		Route:             "/v1/catalog/product-lines/{id}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ObjectType:        constants.ObjectTypeProductLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
			return svc.(ProductLineSvc).UpdateProductLine
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductLine,
			Fields:     []string{"owner", "owner.account", "unit_group"},
		}),
	})
}
