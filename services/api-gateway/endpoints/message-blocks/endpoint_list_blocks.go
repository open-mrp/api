package blockep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to list the account users the caller has blocked.
type ListBlocksRequest struct{}

// Lists the caller's messaging blocks.
type ListBlocksEndpoint struct{}

func (e *ListBlocksEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListBlocksRequest, *apiresource.List[apiresource.MessagingBlock]] {
	return (&apiendpoint.APIEndpoint[*ListBlocksRequest, *apiresource.List[apiresource.MessagingBlock]]{
		Title:               "List Blocks",
		Method:              http.MethodGet,
		ContentType:         "application/json",
		Route:               "/v1/messaging/blocks",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeMessagingBlock,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionRead}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListBlocksRequest) (*apiresource.List[apiresource.MessagingBlock], *apierror.APIError) {
			return svc.(BlockSvc).ListBlocks
		},
		IncludeConfig: blockIncludeConfig(),
	})
}
