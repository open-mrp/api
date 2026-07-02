package searchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to search across resource types for a free-text term.
type SearchRequest struct {
	apiresource.PaginationRequest
	// Filter the search to specific resource types.
	//
	// Omit to search every supported type the caller can read.
	Types []constants.ObjectType `query:"types"`
	// Restrict the search to a single customer by their account ID.
	//
	// When set, only resource types that are safe to expose to a customer are searched (their sales orders, invoices, and shipments), and results are limited to records belonging to that customer.
	Customer *string `query:"customer"`
}

// Search returns lightweight `entity` references matching the query across the resource types the caller can read.
type SearchEndpoint struct{}

func (e *SearchEndpoint) Materialize() *apiendpoint.APIEndpoint[*SearchRequest, *apiresource.List[apiresource.Entity]] {
	return (&apiendpoint.APIEndpoint[*SearchRequest, *apiresource.List[apiresource.Entity]]{
		Title:               "Search",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/core/search",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		AgentTool:           true,
		RequiredPermissions: searchReadPermissions,
		ObjectType:          constants.ObjectTypeEntity,
		ServiceHandler: func(svc any) func(ctx context.Context, req *SearchRequest) (*apiresource.List[apiresource.Entity], *apierror.APIError) {
			return svc.(SearchSvc).Search
		},
	})
}
