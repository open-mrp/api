package itemep

import (
	"context"
	"net/http"

	"google.golang.org/grpc"

	jobep "github.com/open-mrp/api/services/api-gateway/endpoints/jobs"
	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/core"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
)

// The items Set Order Units applies to. Every filter sent must match.
type SetOrderUnitsFilter struct {
	// Only items with these IDs. At most 1,000.
	ItemIDs []string `json:"item_ids,omitzero" validate:"omitempty,max=1000"`
	// Only items sold under one of these product lines.
	ProductLineIDs []string `json:"product_line_ids,omitzero" validate:"omitempty,max=100"`
	// Only items in one of these categories.
	CategoryIDs []string `json:"category_ids,omitzero" validate:"omitempty,max=100"`
	// Only items carrying at least one of these attributes.
	AttributeIDs []string `json:"attribute_ids,omitzero" validate:"omitempty,max=100"`
}

// Request to set the order units of many items.
type SetOrderUnitsRequest struct {
	// The items to change. Send at least one filter; at most 1,000 items may match.
	Filter SetOrderUnitsFilter `json:"filter" validate:"required"`
	// Units the items can be ordered in besides their stocking units. Replaces each item's list; send `[]` to remove them all.
	OrderUnits []ItemOrderUnitInput `json:"order_units" validate:"max=100,dive"`
	// ID of the unit selected by default on orders: the stocking unit or one of `order_units`.
	//
	// Omit to use each item's stocking unit (a service's first order unit).
	DefaultOrderUnitID field.Optional[string] `json:"default_order_unit_id,omitzero"`
}

var sampleSetOrderUnitsRequest = &SetOrderUnitsRequest{
	Filter: SetOrderUnitsFilter{ProductLineIDs: []string{apiresource.SampleProductLineID}},
	OrderUnits: []ItemOrderUnitInput{{
		UnitID:           apiresource.SampleUnitID,
		PortalVisibility: field.Some(constants.CustomerPortalVisibilityVisible),
		Discount:         field.Some(apirequest.DiscountInput{Type: constants.DiscountTypePercentOff, PercentOff: field.Some("5")}),
	}},
}

func (*SetOrderUnitsRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleSetOrderUnitsRequest)
}

// Sets orderable units on many items at once.
//
// Replaces `order_units` and `default_order_unit` on every item matching the filter. The work runs in the background; the response is an async job to poll. Items are written in batches of 100, so a job that fails part-way may have changed some of them; running it again leaves items that already have the requested units as they are.
type SetOrderUnitsEndpoint struct{}

func (e *SetOrderUnitsEndpoint) Materialize() *apiendpoint.APIEndpoint[*SetOrderUnitsRequest, *apiresource.AsyncJob] {
	return (&apiendpoint.APIEndpoint[*SetOrderUnitsRequest, *apiresource.AsyncJob]{
		Title:             "Set Order Units",
		Method:            http.MethodPost,
		Route:             "/v1/catalog/items/actions/set-order-units",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionUpdate}}},
		ObjectType:        constants.ObjectTypeAsyncJob,
		ServiceHandler: func(svc any) func(ctx context.Context, req *SetOrderUnitsRequest) (*apiresource.AsyncJob, *apierror.APIError) {
			return svc.(ItemSvc).SetOrderUnits
		},
		LocationFunc: func(resp *apiresource.AsyncJob) string {
			return "/v1/core/async-jobs/" + resp.ID
		},
	})
}

func (m *itemSvcImpl) SetOrderUnits(ctx context.Context, req *SetOrderUnitsRequest) (*apiresource.AsyncJob, *apierror.APIError) {
	pbReq := &pb.SetItemOrderUnitsRequest{
		ItemIds:            req.Filter.ItemIDs,
		ProductLineIds:     req.Filter.ProductLineIDs,
		CategoryIds:        req.Filter.CategoryIDs,
		AttributeIds:       req.Filter.AttributeIDs,
		OrderUnits:         orderUnitsToProto(req.OrderUnits),
		DefaultOrderUnitId: req.DefaultOrderUnitID.Ptr(),
	}
	resp, apiErr := grpcutil.CallRPC(ctx, itemSvcTracer, "service.items.set_order_units", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.SetItemOrderUnitsResponse, error) {
			return m.coreClient.SetItemOrderUnits(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}
	return jobep.JobFromProto(ctx, resp.GetJob()), nil
}
