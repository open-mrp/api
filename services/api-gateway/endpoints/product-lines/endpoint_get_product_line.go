package productlineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// GetProductLineRequest is the request to retrieve a single product line.
type GetProductLineRequest struct {
	// The ID of the product line to retrieve.
	ProductLineID string `path:"id" validate:"required"`
}

type GetProductLineEndpoint struct{}

func (e *GetProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetProductLineRequest, *apiresource.ProductLine] {
	return &apiendpoint.APIEndpoint[*GetProductLineRequest, *apiresource.ProductLine]{
		Title:             "Get Product Line",
		Description:       "Returns a product line by its ID, including system-owned product lines accessible to the account.",
		Method:            http.MethodGet,
		Route:             "/v1/catalog/product-lines/{id}",
		ContentType:       "application/json",
		Request:           &GetProductLineRequest{},
		Response:          &apiresource.ProductLine{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
			return svc.(ProductLineSvc).GetProductLine
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductLine,
			Fields:     []string{"owner", "owner.account", "unit_group"},
		}),
	}
}
