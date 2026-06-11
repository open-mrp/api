package accountep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to look up an account by portal slug.
type RetrieveAccountBySlugRequest struct {
	// Portal slug.
	Slug string `path:"slug" validate:"required"`
}

// Returns a minimal public profile for the account that owns the given portal slug.
//
// This endpoint does not require authentication; it is intended for customer portal branding lookups.
type RetrieveAccountBySlugEndpoint struct{}

func (e *RetrieveAccountBySlugEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrieveAccountBySlugRequest, *apiresource.PublicAccount] {
	return (&apiendpoint.APIEndpoint[*RetrieveAccountBySlugRequest, *apiresource.PublicAccount]{
		Title:             "Retrieve Account by Slug",
		Method:            http.MethodGet,
		Route:             "/v1/identity/portal-branding/{slug}",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePublicAccount,
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrieveAccountBySlugRequest) (*apiresource.PublicAccount, *apierror.APIError) {
			return svc.(AccountSvc).GetAccountBySlug
		},
	})
}
