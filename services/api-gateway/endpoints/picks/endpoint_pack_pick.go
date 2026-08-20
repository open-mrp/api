package pickep

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

// Request to pack a pick, creating a shipment from the picked lines.
type PackPickRequest struct {
	// Pick ID.
	PickID string `path:"id" validate:"required"`
	// Number of shipping cases to create on the new shipment.
	//
	// Must be at least 1. Cases are numbered sequentially from the shipment number (e.g. `SO-001-1`, `SO-001-2`), and each starts with zero freight weight and freight cost for you to fill in later.
	ShipmentCaseCount int32 `json:"shipment_case_count" validate:"required,gte=1"`
}

var samplePackPickRequest = &PackPickRequest{
	ShipmentCaseCount: 3,
}

func (*PackPickRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(samplePackPickRequest)
}

// Packs a pick, creating a shipment from the picked lines.
//
// Returns `202 Accepted` with a job, because packing writes a shipment, one shipment line per packed pick line, and the requested shipping cases. Poll the job at the returned `Location`; once it reports `completed`, its first result carries the new shipment's `id`, with the shipment line and shipping case ids in `sub_resource_ids`. Every unpacked line with a picked quantity greater than zero is marked as packed and added to a new shipment in `packed` status, which inherits the sales order's carrier, service level, and shipping address. When a sales order line still has outstanding quantity afterward and no unpacked pick line is already open for it, a new zero-quantity pick line is created for the remainder, so packing a partial pick leaves the pick open for the next round. The pick is marked finished only once every one of its lines is packed.
//
// Returns a validation error if no line on the pick has a picked quantity greater than zero.
type PackPickEndpoint struct{}

func (e *PackPickEndpoint) Materialize() *apiendpoint.APIEndpoint[*PackPickRequest, *apiresource.Job] {
	return (&apiendpoint.APIEndpoint[*PackPickRequest, *apiresource.Job]{
		Title:             "Pack Pick",
		Method:            http.MethodPost,
		Route:             "/v1/operations/picks/{id}/actions/pack",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypeJob,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPicks, Action: types.ActionUpdate},
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeJob,
			Fields:     []string{"created_by", "created_by.role"},
		}),
		ServiceHandler: func(svc any) func(ctx context.Context, req *PackPickRequest) (*apiresource.Job, *apierror.APIError) {
			return svc.(PickSvc).PackPick
		},
		LocationFunc: func(resp *apiresource.Job) string {
			return "/v1/core/jobs/" + resp.ID
		},
	})
}
