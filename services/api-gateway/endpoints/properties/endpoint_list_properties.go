package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list properties.
type ListPropertiesRequest struct {
	apiresource.PaginationRequest
}

type ListPropertiesEndpoint struct{}

func (e *ListPropertiesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPropertiesRequest, *apiresource.List[apiresource.Property]] {
	return &apiendpoint.APIEndpoint[*ListPropertiesRequest, *apiresource.List[apiresource.Property]]{
		Title:             "List Properties",
		Description:       "Returns a paginated list of properties for the target account.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/properties",
		Request:           &ListPropertiesRequest{},
		Response:          &apiresource.List[apiresource.Property]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProperty,
			Fields:     []string{"attributes"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPropertiesRequest) (*apiresource.List[apiresource.Property], *apierror.APIError) {
			return svc.(PropertySvc).ListProperties
		},
	}
}
