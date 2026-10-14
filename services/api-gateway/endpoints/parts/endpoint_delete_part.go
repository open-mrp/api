package partep

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

// Request to delete a part.
type DeletePartRequest struct {
	// ID of the part to delete.
	ItemID string `path:"id" validate:"required"`
}

// Deletes a part.
//
// This is a soft delete: the part is marked deleted and no longer returned by other endpoints, but the record is retained. Deleting an already-deleted part returns an error.
type DeletePartEndpoint struct{}

func (e *DeletePartEndpoint) Materialize() *apiendpoint.APIEndpoint[*DeletePartRequest, *object.Deleted] {
	return (&apiendpoint.APIEndpoint[*DeletePartRequest, *object.Deleted]{
		Title:             "Delete Part",
		Method:            http.MethodDelete,
		ContentType:       "application/json",
		Route:             "/v1/catalog/parts/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainParts, Action: types.ActionDelete}}, Counterparty: authz.Counterparties(types.ActionUpdate)},
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *DeletePartRequest) (*object.Deleted, *apierror.APIError) {
			return apiresource.Deleted(svc.(PartSvc).DeletePart, constants.ObjectTypePart, "id")
		},
	})
}
