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

// Request to list who may sign a batch off.
type ListEligibleSignersRequest struct {
	// ID of the batch.
	BatchID string `path:"id" validate:"required"`
}

// Returns who may sign a batch off in its department, and in which capacities.
//
// Admins, supervisors holding `batch_releases` (only those in the batch's department, or with no department), and quality inspectors holding `quality_releases`. When the department needs someone other than the operator to sign, the operator who scanned the batch is left out. Ordered by name; not paginated.
type ListEligibleSignersEndpoint struct{}

func (e *ListEligibleSignersEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListEligibleSignersRequest, *apiresource.List[apiresource.EligibleSigner]] {
	return (&apiendpoint.APIEndpoint[*ListEligibleSignersRequest, *apiresource.List[apiresource.EligibleSigner]]{
		Title:             "List Eligible Signers",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/batches/{id}/eligible-signers",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeEligibleSigner,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainBatches, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListEligibleSignersRequest) (*apiresource.List[apiresource.EligibleSigner], *apierror.APIError) {
			return svc.(BatchReleaseSvc).ListEligibleSigners
		},
	})
}
