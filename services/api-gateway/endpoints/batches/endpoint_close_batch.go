package batchep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to close a batch.
type CloseBatchRequest struct {
	// Batch ID.
	BatchID string `json:"batch_id" validate:"required"`
}

var sampleCloseBatchRequest = &CloseBatchRequest{
	BatchID: apiresource.SampleBatchID,
}

func (*CloseBatchRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCloseBatchRequest)
}

// Closes a batch so it can no longer be scanned or advanced through production.
//
// Use this to finish a batch whose remaining quantity will not be produced, for example when the floor stops short of the planned output. Batches also close on their own when they reach the last production step, when they are moved or merged into a downstream batch, and when everything split off them accounts for their whole quantity. A closed batch cannot be reopened.
type CloseBatchEndpoint struct{}

func (e *CloseBatchEndpoint) Materialize() *apiendpoint.APIEndpoint[*CloseBatchRequest, *apiresource.Batch] {
	return (&apiendpoint.APIEndpoint[*CloseBatchRequest, *apiresource.Batch]{
		Title:               "Close Batch",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/operations/batches/actions/close",
		SuccessStatusCode:   http.StatusOK,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainBatches, Action: types.ActionDelete}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CloseBatchRequest) (*apiresource.Batch, *apierror.APIError) {
			return svc.(BatchSvc).CloseBatch
		},
	})
}
