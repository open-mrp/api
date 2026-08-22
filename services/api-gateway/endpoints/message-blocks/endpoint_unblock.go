package blockep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to remove a block the caller created.
type UnblockRequest struct {
	// The account user to unblock.
	//
	// This is the ID of the blocked account user, not the block record's own ID.
	BlockedAccountUserID string `path:"id" validate:"required"`
}

// Lifts a block you placed on another user, letting the two of you message each other again.
//
// Only your own block is removed: if the other person has also blocked you, direct messages between you stay blocked. Unblocking someone you have not blocked succeeds and changes nothing.
type UnblockEndpoint struct{}

func (e *UnblockEndpoint) Materialize() *apiendpoint.APIEndpoint[*UnblockRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*UnblockRequest, *apiresource.EmptyResource]{
		Title:               "Unblock User",
		Method:              http.MethodDelete,
		ContentType:         "application/json",
		Route:               "/v1/messaging/blocks/{id}",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UnblockRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(BlockSvc).Unblock
		},
	})
}
