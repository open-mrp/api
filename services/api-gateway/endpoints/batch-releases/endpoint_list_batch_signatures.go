package batchreleaseep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list a batch's signatures.
type ListBatchSignaturesRequest struct {
	// ID of the batch.
	BatchID string `path:"id" validate:"required"`
}

// Returns every signature a batch has received, newest first.
//
// Signatures stay after the batch's scan is undone. A batch collects a handful at most, so the list isn't paginated.
type ListBatchSignaturesEndpoint struct{}

func (e *ListBatchSignaturesEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListBatchSignaturesRequest, *apiresource.List[apiresource.BatchSignature]] {
	return (&apiendpoint.APIEndpoint[*ListBatchSignaturesRequest, *apiresource.List[apiresource.BatchSignature]]{
		Title:             "List Batch Signatures",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/batches/{id}/signatures",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeBatchSignature,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainBatches, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListBatchSignaturesRequest) (*apiresource.List[apiresource.BatchSignature], *apierror.APIError) {
			return svc.(BatchReleaseSvc).ListBatchSignatures
		},
	})
}
