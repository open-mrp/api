package itemep

import (
	"context"
	"net/http"

	"google.golang.org/grpc"

	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
)

// Request to change an item's stocking unit.
type ChangeStockingUnitRequest struct {
	// Item ID.
	ItemID string `path:"id" validate:"required"`
	// ID of the unit the item will be stocked in.
	StockingUnitID string `json:"stocking_unit_id" validate:"required"`
	// How the numbers recorded in the old unit are treated: `convert` or `relabel`.
	//
	// `convert` restates inventory, lots, costs and prices into the new unit, which must measure the same dimension, and keeps the old unit as an order unit. `relabel` keeps every number and only changes the unit, to fix one entered in error; order units of another dimension than the new unit are removed and listed on the async job's result.
	Mode constants.ChangeStockingUnitMode `json:"mode" validate:"required"`
}

var sampleChangeStockingUnitRequest = &ChangeStockingUnitRequest{
	ItemID:         apiresource.SampleItemID,
	StockingUnitID: apiresource.SampleUnitID,
	Mode:           constants.ChangeStockingUnitModeConvert,
}

func (*ChangeStockingUnitRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleChangeStockingUnitRequest)
}

// Changes the unit an item is stocked in.
//
// `convert` restates inventory, lots and costs into the new unit of the same dimension. `relabel` keeps the numbers and only changes the unit, to fix one entered in error. The work runs in the background; the response is an async job to poll.
type ChangeStockingUnitEndpoint struct{}

func (e *ChangeStockingUnitEndpoint) Materialize() *apiendpoint.APIEndpoint[*ChangeStockingUnitRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*ChangeStockingUnitRequest, *apiresource.AsyncJob]{
		Title:             "Change Stocking Unit",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/items/{id}/actions/change-stocking-unit",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeAsyncJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ChangeStockingUnitRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(ItemSvc).ChangeStockingUnit
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}

func (m *itemSvcImpl) ChangeStockingUnit(ctx context.Context, req *ChangeStockingUnitRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.change_stocking_unit", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ChangeItemStockingUnitResponse, error) {
			return m.coreClient.ChangeItemStockingUnit(ctx, &pb.ChangeItemStockingUnitRequest{
				Id:             req.ItemID,
				StockingUnitId: req.StockingUnitID,
				Mode:           string(req.Mode),
			}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return jobep.JobFromProto(ctx, resp.GetJob()), nil
}
