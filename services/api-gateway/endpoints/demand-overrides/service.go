package demandoverridesep

import (
	"context"
	"fmt"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DemandOverridesSvc interface {
	ListDemandOverrideTypes(ctx context.Context, req *ListDemandOverrideTypesRequest) (*apiresource.List[apiresource.DemandOverrideType], *apierror.APIError)
	ListDemandOverrides(ctx context.Context, req *ListDemandOverridesRequest) (*apiresource.List[apiresource.DemandOverride], *apierror.APIError)
	GetDemandOverride(ctx context.Context, req *RetrieveDemandOverrideRequest) (*apiresource.DemandOverride, *apierror.APIError)
	CreateDemandOverride(ctx context.Context, req *CreateDemandOverrideRequest) (*apiresource.DemandOverride, *apierror.APIError)
	UpdateDemandOverride(ctx context.Context, req *UpdateDemandOverrideRequest) (*apiresource.DemandOverride, *apierror.APIError)
	DeleteDemandOverride(ctx context.Context, req *DeleteDemandOverrideRequest) (*apiresource.EmptyResource, *apierror.APIError)
}

type DemandOverridesSvcConfig struct {
	// CoreClient (required) is the core-service demand-override gRPC client.
	CoreClient pb.CoreDemandOverrideServiceClient
}

type demandOverridesSvcImpl struct {
	coreClient pb.CoreDemandOverrideServiceClient
}

var demandOverridesEpSvcTracer = tracing.GetTracer("api-gateway.endpoints.demand-overrides.service")

func (c *DemandOverridesSvcConfig) validate() error {
	if c.CoreClient == nil {
		return fmt.Errorf("demand overrides endpoint service: core client is required")
	}
	return nil
}

func NewDemandOverridesSvc(config *DemandOverridesSvcConfig) DemandOverridesSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &demandOverridesSvcImpl{
		coreClient: config.CoreClient,
	}
}

func (m *demandOverridesSvcImpl) ListDemandOverrideTypes(ctx context.Context, req *ListDemandOverrideTypesRequest) (*apiresource.List[apiresource.DemandOverrideType], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, demandOverridesEpSvcTracer, "service.demand_override.list_types", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListDemandOverrideTypesResponse, error) {
			return m.coreClient.ListDemandOverrideTypes(ctx, &pb.ListDemandOverrideTypesRequest{}, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	types := make([]apiresource.DemandOverrideType, len(resp.Types))
	for i, t := range resp.Types {
		types[i] = DemandOverrideTypeFromProto(t)
	}

	return apiresource.NewList(types, apiresource.PageInfo{}), nil
}

func (m *demandOverridesSvcImpl) ListDemandOverrides(ctx context.Context, req *ListDemandOverridesRequest) (*apiresource.List[apiresource.DemandOverride], *apierror.APIError) {
	pbReq := &pb.ListDemandOverridesRequest{
		Cursor:            req.Cursor,
		Limit:             req.Limit,
		ScopeCodes:        overrideEnumStrings(req.ScopeTypes),
		ScopeRefIds:       req.ScopeRefIDs,
		OverrideTypeCodes: overrideEnumStrings(req.Adjustments),
		IsActive:          activationFilter(req.Statuses),
		Query:             req.Query,
		PeriodStart:       req.PeriodStart,
		PeriodEnd:         req.PeriodEnd,
	}

	resp, apiErr := grpcutil.CallRPC(ctx, demandOverridesEpSvcTracer, "service.demand_override.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListDemandOverridesResponse, error) {
			return m.coreClient.ListDemandOverrides(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	overrides := make([]apiresource.DemandOverride, len(resp.Overrides))
	creators := make([]*apiresource.Actor, 0, len(resp.Overrides))
	for i, o := range resp.Overrides {
		overrides[i] = DemandOverrideFromProto(o)
		creators = append(creators, StashDemandOverrideMeta(ctx, meta, o))
	}
	hydrateCreators(ctx, creators)

	return apiresource.NewList(overrides, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *demandOverridesSvcImpl) GetDemandOverride(ctx context.Context, req *RetrieveDemandOverrideRequest) (*apiresource.DemandOverride, *apierror.APIError) {
	pbReq := &pb.GetDemandOverrideRequest{Id: req.DemandOverrideID}

	resp, apiErr := grpcutil.CallRPC(ctx, demandOverridesEpSvcTracer, "service.demand_override.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetDemandOverrideResponse, error) {
			return m.coreClient.GetDemandOverride(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := DemandOverrideFromProto(resp.Override)
	hydrateCreators(ctx, []*apiresource.Actor{StashDemandOverrideMeta(ctx, meta, resp.Override)})
	return &result, nil
}

func (m *demandOverridesSvcImpl) CreateDemandOverride(ctx context.Context, req *CreateDemandOverrideRequest) (*apiresource.DemandOverride, *apierror.APIError) {
	pbReq := &pb.CreateDemandOverrideRequest{
		ScopeCode:        string(req.ScopeType),
		ScopeRefId:       req.ScopeRefID,
		PeriodStartDate:  timestamppb.New(req.PeriodStartsAt),
		PeriodEndDate:    timestamppb.New(req.PeriodEndsAt),
		OverrideTypeCode: string(req.Adjustment),
		Value:            req.Value,
		UnitId:           req.UnitID.Ptr(),
		ReasonCode:       enumPtr(req.Reason.Ptr()),
		Note:             req.Note.Ptr(),
		IsActive:         req.Active.Ptr(),
	}
	if effectiveAt := req.EffectiveAt.Ptr(); effectiveAt != nil {
		pbReq.EffectiveFrom = timestamppb.New(*effectiveAt)
	}
	if expiresAt := req.ExpiresAt.Ptr(); expiresAt != nil {
		pbReq.ExpiresAt = timestamppb.New(*expiresAt)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, demandOverridesEpSvcTracer, "service.demand_override.create", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.CreateDemandOverrideResponse, error) {
			return m.coreClient.CreateDemandOverride(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := DemandOverrideFromProto(resp.Override)
	hydrateCreators(ctx, []*apiresource.Actor{StashDemandOverrideMeta(ctx, meta, resp.Override)})
	return &result, nil
}

func (m *demandOverridesSvcImpl) UpdateDemandOverride(ctx context.Context, req *UpdateDemandOverrideRequest) (*apiresource.DemandOverride, *apierror.APIError) {
	// Clearable enum: map Clearable[DemandOverrideReason] to a StringPatch (clear vs set vs leave).
	var reasonPatch *pb.StringPatch
	switch {
	case req.Reason.IsClear():
		reasonPatch = &pb.StringPatch{Clear: true}
	case req.Reason.IsSet():
		v, _ := req.Reason.Value()
		s := string(v)
		reasonPatch = &pb.StringPatch{Value: &s}
	}

	pbReq := &pb.UpdateDemandOverrideRequest{
		Id:               req.DemandOverrideID,
		OverrideTypeCode: enumPtr(req.Adjustment.Ptr()),
		Value:            req.Value.Ptr(),
		// Clearable nullable fields → *Patch (clear / set / leave). Clearing expires_at makes the override permanent again.
		UnitId:     field.StringClearableToProto(req.UnitID),
		ReasonCode: reasonPatch,
		Note:       field.StringClearableToProto(req.Note),
		ExpiresAt:  field.TimestampClearableToProto(req.ExpiresAt),
		IsActive:   req.Active.Ptr(),
	}
	if periodStart := req.PeriodStartsAt.Ptr(); periodStart != nil {
		pbReq.PeriodStartDate = timestamppb.New(*periodStart)
	}
	if periodEnd := req.PeriodEndsAt.Ptr(); periodEnd != nil {
		pbReq.PeriodEndDate = timestamppb.New(*periodEnd)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, demandOverridesEpSvcTracer, "service.demand_override.update", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.UpdateDemandOverrideResponse, error) {
			return m.coreClient.UpdateDemandOverride(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := DemandOverrideFromProto(resp.Override)
	hydrateCreators(ctx, []*apiresource.Actor{StashDemandOverrideMeta(ctx, meta, resp.Override)})
	return &result, nil
}

func (m *demandOverridesSvcImpl) DeleteDemandOverride(ctx context.Context, req *DeleteDemandOverrideRequest) (*apiresource.EmptyResource, *apierror.APIError) {
	pbReq := &pb.DeleteDemandOverrideRequest{Id: req.DemandOverrideID}

	_, apiErr := grpcutil.CallRPC(ctx, demandOverridesEpSvcTracer, "service.demand_override.delete", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*emptypb.Empty, error) {
			return m.coreClient.DeleteDemandOverride(ctx, pbReq, opts...)
		})
	if apiErr != nil {
		return nil, apiErr
	}

	return &apiresource.EmptyResource{}, nil
}

// DemandOverrideTypeFromProto maps a core DemandOverrideTypeInfo to the API resource.
func DemandOverrideTypeFromProto(info *pb.DemandOverrideTypeInfo) apiresource.DemandOverrideType {
	return apiresource.DemandOverrideType{
		ID:        info.Id,
		Object:    constants.ObjectTypeDemandOverrideType,
		Code:      constants.DemandOverrideAdjustment(info.Code),
		Name:      info.Name,
		CreatedAt: grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt: grpcutil.TimestampToTime(info.UpdatedAt),
	}
}

// DemandOverrideFromProto maps a core DemandOverrideInfo to the API resource. The scope, unit and created_by expandables are left nil; pair with StashDemandOverrideMeta so they resolve on ?include=.
func DemandOverrideFromProto(info *pb.DemandOverrideInfo) apiresource.DemandOverride {
	o := apiresource.DemandOverride{
		ID:             info.Id,
		Object:         constants.ObjectTypeDemandOverride,
		ScopeType:      constants.DemandOverrideScope(info.ScopeCode),
		PeriodStartsAt: grpcutil.TimestampToTime(info.PeriodStartDate),
		PeriodEndsAt:   grpcutil.TimestampToTime(info.PeriodEndDate),
		Adjustment:     constants.DemandOverrideAdjustment(info.OverrideTypeCode),
		Value:          info.Value,
		Reason:         constants.DemandOverrideReasonPtr(info.ReasonCode),
		Note:           info.Note,
		EffectiveAt:    grpcutil.TimestampToTime(info.EffectiveFrom),
		Status:         constants.ActivationStatusOf(info.IsActive),
		CreatedAt:      grpcutil.TimestampToTime(info.CreatedAt),
		UpdatedAt:      grpcutil.TimestampToTime(info.UpdatedAt),
	}
	o.ExpiresAt = grpcutil.TimestampToTimePtr(info.ExpiresAt)
	return o
}

// StashDemandOverrideMeta stashes the reference data for the expandable sub-resources. The scope is polymorphic, so it resolves to an Entity rather than to a typed sub-resource: an include that can only ever populate for half the rows is not an include. The creator is stored as a bare identity-actor id, so its Actor is built from the id's prefix and preheated — LoadActors has no backing store. The built actor is returned (nil when absent) so the caller can batch-hydrate display names.
func StashDemandOverrideMeta(ctx context.Context, meta *resourcekit.LoadMeta, info *pb.DemandOverrideInfo) *apiresource.Actor {
	if info == nil {
		return nil
	}
	if entity := demandOverrideScopeEntity(info); entity != nil {
		meta.Set(constants.ObjectTypeDemandOverride, info.Id, "scope", entity)
	}
	if info.UnitId != nil && *info.UnitId != "" {
		meta.Set(constants.ObjectTypeDemandOverride, info.Id, "unit_id", *info.UnitId)
	}
	actor := resourceloaders.ActorRefFromID(info.CreatedById)
	if actor != nil {
		meta.Set(constants.ObjectTypeDemandOverride, info.Id, "created_by_id", actor.ID)
		resourcekit.PreheatCache(ctx, constants.ObjectTypeActor, actor.ID, actor)
	}
	return actor
}

// hydrateCreators fills the creators' display names + handles. A no-op unless the caller expanded created_by — the only case where the names are rendered — avoiding needless loader round-trips otherwise.
func hydrateCreators(ctx context.Context, creators []*apiresource.Actor) {
	if !resourcekit.RequestedIncludeSet(ctx)["created_by"] {
		return
	}
	resourceloaders.HydrateIdentityActorNames(ctx, creators)
}

// demandOverrideScopeEntity maps the scope code onto the object type it references.
func demandOverrideScopeEntity(info *pb.DemandOverrideInfo) *apiresource.Entity {
	if info.ScopeRefId == "" {
		return nil
	}
	var scopeType constants.ObjectType
	switch info.ScopeCode {
	case string(constants.DemandOverrideScopeItem):
		scopeType = constants.ObjectTypeItem
	case string(constants.DemandOverrideScopeProductLine):
		scopeType = constants.ObjectTypeProductLine
	default:
		return nil
	}
	return apiresource.NewEntity(info.ScopeRefId, scopeType, info.ScopeName, info.ScopeHandle)
}

// enumPtr narrows a typed-enum pointer to the plain string pointer the proto layer uses. The storage vocabulary stays untyped; the typing lives at the API boundary.
func enumPtr[T ~string](v *T) *string {
	if v == nil {
		return nil
	}
	s := string(*v)
	return &s
}

func overrideEnumStrings[T ~string](values []T) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

// activationFilter collapses the status filter onto the stored boolean. Asking for both states is the same as not filtering, so it returns nil rather than an impossible "active AND inactive" predicate.
func activationFilter(statuses []constants.ActivationStatus) *bool {
	var wantActive, wantInactive bool
	for _, status := range statuses {
		switch status {
		case constants.ActivationStatusActive:
			wantActive = true
		case constants.ActivationStatusInactive:
			wantInactive = true
		}
	}
	if wantActive == wantInactive {
		return nil
	}
	return &wantActive
}
