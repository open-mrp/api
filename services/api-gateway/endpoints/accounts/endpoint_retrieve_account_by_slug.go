package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to look up an account by portal slug.
type RetrieveAccountBySlugRequest struct {
	// Portal slug.
	Slug string `path:"slug" validate:"required"`
}

type RetrieveAccountBySlugEndpoint struct{}

func (e *RetrieveAccountBySlugEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAccountBySlugRequest, *apiresource.PublicAccount] {
	return &apiendpoint.APIEndpoint[*RetrieveAccountBySlugRequest, *apiresource.PublicAccount]{
		Title:             "Retrieve Account by Slug",
		Description:       "Returns a public account by portal slug. Unauthenticated.",
		Method:            http.MethodGet,
		Route:             "/v1/identity/portal-branding/{slug}",
		ContentType:       "application/json",
		Request:           &RetrieveAccountBySlugRequest{},
		Response:          &apiresource.PublicAccount{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAccountBySlugRequest) (*apiresource.PublicAccount, *apierror.APIError) {
			return svc.(AccountSvc).GetAccountBySlug
		},
	}
}
