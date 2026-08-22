package resourceloaders

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
)

var demandOverrideLoaderTracer = tracing.GetTracer("api-gateway.resourceloaders.demand_override")

func LoadDemandOverrides(ctx context.Context, ids []string) (map[string]any, *apierror.APIError) {
	if len(ids) == 0 {
		return nil, nil
	}
	resp, apiErr := grpcutil.CallRPC(ctx, demandOverrideLoaderTracer, "loader.demand_overrides.batch_get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.BatchGetDemandOverridesByIDsResponse, error) {
			return demandOverrideClient.BatchGetDemandOverridesByIDs(ctx, &pb.BatchGetDemandOverridesByIDsRequest{Ids: ids}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	out := make(map[string]any, len(resp.Overrides))
	for _, o := range resp.Overrides {
		out[o.Id] = DemandOverrideFromProto(o)
		stashDemandOverrideRefs(ctx, meta, o)
	}
	return out, nil
}

func DemandOverrideFromProto(o *pb.DemandOverrideInfo) *apiresource.DemandOverride {
	out := &apiresource.DemandOverride{
		ID:             o.Id,
		Object:         constants.ObjectTypeDemandOverride,
		ScopeType:      constants.DemandOverrideScope(o.ScopeCode),
		PeriodStartsAt: grpcutil.TimestampToTime(o.PeriodStartDate),
		PeriodEndsAt:   grpcutil.TimestampToTime(o.PeriodEndDate),
		Adjustment:     constants.DemandOverrideAdjustment(o.OverrideTypeCode),
		Value:          o.Value,
		Reason:         constants.DemandOverrideReasonPtr(o.ReasonCode),
		Note:           o.Note,
		EffectiveAt:    grpcutil.TimestampToTime(o.EffectiveFrom),
		Status:         constants.ActivationStatusOf(o.IsActive),
		CreatedAt:      grpcutil.TimestampToTime(o.CreatedAt),
		UpdatedAt:      grpcutil.TimestampToTime(o.UpdatedAt),
	}
	out.ExpiresAt = grpcutil.TimestampToTimePtr(o.ExpiresAt)
	return out
}

// stashDemandOverrideRefs stashes the reference data for the expandable scope, unit and
// created_by sub-resources. The creator is a bare identity-actor id, so its Actor is
// built from the id's prefix and preheated — LoadActors has no backing store.
func stashDemandOverrideRefs(ctx context.Context, meta *resourcekit.LoadMeta, o *pb.DemandOverrideInfo) {
	if entity := demandOverrideScopeEntity(o); entity != nil {
		meta.Set(constants.ObjectTypeDemandOverride, o.Id, "scope", entity)
	}
	if o.UnitId != nil && *o.UnitId != "" {
		meta.Set(constants.ObjectTypeDemandOverride, o.Id, "unit_id", *o.UnitId)
	}
	if actor := ActorRefFromID(o.CreatedById); actor != nil {
		meta.Set(constants.ObjectTypeDemandOverride, o.Id, "created_by_id", actor.ID)
		resourcekit.PreheatCache(ctx, constants.ObjectTypeActor, actor.ID, actor)
	}
}

// demandOverrideScopeEntity maps the polymorphic scope onto the object type it references.
func demandOverrideScopeEntity(o *pb.DemandOverrideInfo) *apiresource.Entity {
	if o.ScopeRefId == "" {
		return nil
	}
	var scopeType constants.ObjectType
	switch o.ScopeCode {
	case string(constants.DemandOverrideScopeItem):
		scopeType = constants.ObjectTypeItem
	case string(constants.DemandOverrideScopeProductLine):
		scopeType = constants.ObjectTypeProductLine
	default:
		return nil
	}
	return apiresource.NewEntity(o.ScopeRefId, scopeType, o.ScopeName, o.ScopeHandle)
}
