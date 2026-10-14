package blockep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to remove a block the caller created.
type UnblockRequest struct {
	// The account user to unblock.
	//
	// This is the ID of the blocked account user, not the block record's own ID.
	BlockedAccountUserID string `path:"id" validate:"required"`
}

// Removes a block you placed on another user, letting the two of you message each other again.
//
// Only your own block is removed: if the other person has also blocked you, direct messages between you stay blocked. Unblocking someone you have not blocked succeeds and changes nothing.
type UnblockEndpoint struct{}

func (e *UnblockEndpoint) Materialize() *apiendpoint.APIEndpoint[*UnblockRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*UnblockRequest, *object.Deleted]{
		Title:             "Unblock User",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/messaging/blocks/{id}",
		SuccessStatusCode: http.StatusOK,
		ObjectType:        constants.ObjectTypeMessagingBlock,
		Public:            true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionDelete}}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UnblockRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(BlockSvc).Unblock, constants.ObjectTypeMessagingBlock, "id")
		},
	})
}
