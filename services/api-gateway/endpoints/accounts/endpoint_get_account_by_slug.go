package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to look up an account by portal slug.
type GetAccountBySlugRequest struct {
	// Portal slug.
	Slug string `path:"slug" validate:"required"`
}

type GetAccountBySlugEndpoint struct{}

func (e *GetAccountBySlugEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetAccountBySlugRequest, *apiresource.PublicAccount] {
	return &apiendpoint.APIEndpoint[*GetAccountBySlugRequest, *apiresource.PublicAccount]{
		Title:             "Get Account by Slug",
		Description:       "Returns a public account by portal slug. Unauthenticated.",
		Method:            http.MethodGet,
		Route:             "/v1/identity/portal-branding/{slug}",
		ContentType:       "application/json",
		Request:           &GetAccountBySlugRequest{},
		Response:          &apiresource.PublicAccount{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetAccountBySlugRequest) (*apiresource.PublicAccount, *apierror.APIError) {
			return svc.(AccountSvc).GetAccountBySlug
		},
	}
}
