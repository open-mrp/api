package blockep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to block another account user from messaging the caller.
type BlockRequest struct {
	// The account user to block.
	BlockedAccountUserID string `json:"blocked_account_user_id" validate:"required"`
}

var sampleBlockRequest = &BlockRequest{
	BlockedAccountUserID: apiresource.SampleAccountUserID,
}

func (*BlockRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleBlockRequest)
}

// Blocks an account user (prevents DMs in both directions).
type BlockEndpoint struct{}

func (e *BlockEndpoint) Materialize() *apiendpoint.APIEndpoint[*BlockRequest, *apiresource.MessagingBlock] {
	return (&apiendpoint.APIEndpoint[*BlockRequest, *apiresource.MessagingBlock]{
		Title:               "Block User",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/messaging/blocks",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeMessagingBlock,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainMessaging, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *BlockRequest) (*apiresource.MessagingBlock, *apierror.APIError) {
			return svc.(BlockSvc).Block
		},
		IncludeConfig: blockIncludeConfig(),
	})
}
