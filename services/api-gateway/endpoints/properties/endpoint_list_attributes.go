package propertyep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// ListAttributesRequest is the request to list attributes for a property.
type ListAttributesRequest struct {
	apiresource.PaginationRequest
	// The ID of the property to list attributes for.
	PropertyID string `path:"property_id" validate:"required"`
}

type ListAttributesEndpoint struct{}

func (e *ListAttributesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListAttributesRequest, *apiresource.List[apiresource.Attribute]] {
	return &apiendpoint.APIEndpoint[*ListAttributesRequest, *apiresource.List[apiresource.Attribute]]{
		Title:             "List Attributes",
		Description:       "Returns a paginated list of attributes for a property.",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/properties/{property_id}/attributes",
		Request:           &ListAttributesRequest{},
		Response:          &apiresource.List[apiresource.Attribute]{},
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListAttributesRequest) (*apiresource.List[apiresource.Attribute], *apierror.APIError) {
			return svc.(PropertySvc).ListAttributes
		},
	}
}
