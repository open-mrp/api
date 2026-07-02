package auditeventsep

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/augno/api/services/api-gateway/internal/domain"
	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/platform"
	"github.com/augno/api/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuditEventSvc interface {
	ListAuditEvents(ctx context.Context, req *ListAuditEventsRequest) (*apiresource.List[apiresource.AuditEvent], *apierror.APIError)
	ListAuditEventResourceTypes(ctx context.Context, req *ListAuditEventResourceTypesRequest) (*apiresource.List[constants.ObjectType], *apierror.APIError)
	GetAuditEvent(ctx context.Context, req *RetrieveAuditEventRequest) (*apiresource.AuditEvent, *apierror.APIError)
}

type AuditEventSvcConfig struct {
	// AuditClient (required) is the platform-service audit gRPC client.
	AuditClient pb.AuditServiceClient
}

type auditEventSvcImpl struct {
	auditClient pb.AuditServiceClient
}

var auditEventSvcTracer = tracing.GetTracer("api-gateway.endpoints.audit_events.service")

func (c *AuditEventSvcConfig) validate() error {
	if c.AuditClient == nil {
		return fmt.Errorf("audit events endpoint service: audit client is required")
	}
	return nil
}

func NewAuditEventSvc(config *AuditEventSvcConfig) AuditEventSvc {
	if err := config.validate(); err != nil {
		panic(err)
	}

	return &auditEventSvcImpl{
		auditClient: config.AuditClient,
	}
}

func (m *auditEventSvcImpl) ListAuditEvents(ctx context.Context, req *ListAuditEventsRequest) (*apiresource.List[apiresource.AuditEvent], *apierror.APIError) {
	actorTypes := make([]string, len(req.ActorTypes))
	for i, at := range req.ActorTypes {
		actorTypes[i] = string(at)
	}

	pbReq := &pb.ListAuditEventsRequest{
		ResourceTypes:    stringsFromObjectTypes(req.ResourceTypes),
		ResourceIds:      req.ResourceIDs,
		ActorIds:         req.ActorIDs,
		ActorTypes:       actorTypes,
		Actions:          stringsFromActions(req.Actions),
		ActorAccountIds:  req.ActorAccountIDs,
		TargetAccountIds: req.TargetAccountIDs,
		Query:            req.Query,
		Cursor:           req.Cursor,
		Limit:            req.Limit,
		Includes:         resourcekit.FilterIncludes(ctx, "actor", "changes", "metadata"),
	}

	if req.StartDate != nil && !req.StartDate.IsZero() {
		pbReq.StartDate = timestamppb.New(*req.StartDate)
	}
	if req.EndDate != nil && !req.EndDate.IsZero() {
		pbReq.EndDate = timestamppb.New(*req.EndDate)
	}

	resp, apiErr := grpcutil.CallRPC(ctx, auditEventSvcTracer, "service.audit_events.list", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAuditEventsResponse, error) {
			return m.auditClient.ListAuditEvents(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	if resp == nil {
		return apiresource.NewList[apiresource.AuditEvent](nil, apiresource.PageInfo{}), nil
	}

	meta := resourcekit.GetLoadMeta(ctx)
	events := make([]apiresource.AuditEvent, len(resp.AuditEvents))
	var agentActors []*apiresource.Actor
	for i, ev := range resp.AuditEvents {
		events[i] = auditEventFromProto(ev)
		if actor := stashAuditEventMeta(ctx, meta, ev); actor != nil && actor.Type == constants.ActorTypeAgent {
			agentActors = append(agentActors, actor)
		}
	}
	hydrateAgentActors(ctx, agentActors)

	return apiresource.NewList(events, grpcutil.MapProtoPageInfo(ctx, resp.PageInfo)), nil
}

func (m *auditEventSvcImpl) ListAuditEventResourceTypes(ctx context.Context, _ *ListAuditEventResourceTypesRequest) (*apiresource.List[constants.ObjectType], *apierror.APIError) {
	resp, apiErr := grpcutil.CallRPC(ctx, auditEventSvcTracer, "service.audit_events.list_resource_types", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.ListAuditEventResourceTypesResponse, error) {
			return m.auditClient.ListAuditEventResourceTypes(ctx, &pb.ListAuditEventResourceTypesRequest{}, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	if resp == nil {
		return apiresource.NewList[constants.ObjectType](nil, apiresource.PageInfo{}), nil
	}

	types := make([]constants.ObjectType, len(resp.ResourceTypes))
	for i, rt := range resp.ResourceTypes {
		types[i] = constants.ObjectType(rt)
	}
	return apiresource.NewList(types, apiresource.PageInfo{}), nil
}

func (m *auditEventSvcImpl) GetAuditEvent(ctx context.Context, req *RetrieveAuditEventRequest) (*apiresource.AuditEvent, *apierror.APIError) {
	pbReq := &pb.GetAuditEventRequest{
		Id:       req.ID,
		Includes: resourcekit.FilterIncludes(ctx, "actor", "changes", "metadata"),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, auditEventSvcTracer, "service.audit_events.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAuditEventResponse, error) {
			return m.auditClient.GetAuditEvent(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	meta := resourcekit.GetLoadMeta(ctx)
	result := auditEventFromProto(resp.AuditEvent)
	if actor := stashAuditEventMeta(ctx, meta, resp.AuditEvent); actor != nil && actor.Type == constants.ActorTypeAgent {
		hydrateAgentActors(ctx, []*apiresource.Actor{actor})
	}
	return &result, nil
}

// hydrateAgentActors fills in agent actors' display name + handle from agent-service.
// Unlike user/api_key actors — whose names are joined in platform-service — agent
// definitions live in a separate datastore, so their names must be resolved here.
// Best-effort and a no-op unless the caller expanded the actor (the only case where
// the name is rendered), avoiding a needless agent-service round-trip otherwise.
func hydrateAgentActors(ctx context.Context, actors []*apiresource.Actor) {
	if len(actors) == 0 || !resourcekit.RequestedIncludeSet(ctx)["actor"] {
		return
	}
	resourceloaders.HydrateActorNames(ctx, actors)
}

func auditEventFromProto(ev *pb.AuditEventInfo) apiresource.AuditEvent {
	if ev == nil {
		return apiresource.AuditEvent{}
	}

	return apiresource.AuditEvent{
		ID:             ev.Id,
		Object:         constants.ObjectTypeAuditEvent,
		Action:         constants.AuditAction(ev.Action),
		ResourceType:   constants.ObjectType(ev.ResourceType),
		ResourceID:     ev.ResourceId,
		IdempotencyKey: stringPtrFromOptional(ev.IdempotencyKey),
		SourceIP:       stringPtrFromOptional(ev.SourceIp),
		OccurredAt:     grpcutil.TimestampToTime(ev.OccurredAt),
		CreatedAt:      grpcutil.TimestampToTime(ev.CreatedAt),
	}
}

// stashAuditEventMeta stashes the event's expandable sub-resources into the load
// meta and returns the event's actor (or nil) so the caller can hydrate agent
// actors' names from agent-service, mirroring the request-log path.
func stashAuditEventMeta(ctx context.Context, meta *resourcekit.LoadMeta, ev *pb.AuditEventInfo) *apiresource.Actor {
	if ev == nil {
		return nil
	}

	var actor *apiresource.Actor
	if ev.Actor != nil {
		var name *string
		if ev.Actor.Name != nil && *ev.Actor.Name != "" {
			n := *ev.Actor.Name
			name = &n
		}
		actor = apiresource.NewActor(
			ev.Actor.Id,
			constants.ActorType(ev.Actor.ActorType),
			name,
			stringPtrFromOptional(ev.Actor.Handle),
		)
		resourcekit.PreheatCache(ctx, constants.ObjectTypeActor, actor.ID, actor)
		meta.Set(constants.ObjectTypeAuditEvent, ev.Id, "actor", actor)
	}

	if ev.AccountId != nil && ev.AccountName != nil {
		account := &apiresource.Account{
			ID:     *ev.AccountId,
			Object: constants.ObjectTypeAccount,
			Name:   *ev.AccountName,
		}
		if ev.AccountCreatedAt != nil {
			account.CreatedAt = ev.AccountCreatedAt.AsTime()
		}
		if ev.AccountUpdatedAt != nil {
			account.UpdatedAt = ev.AccountUpdatedAt.AsTime()
		}
		meta.Set(constants.ObjectTypeAuditEvent, ev.Id, "account", account)
	}

	if len(ev.Changes) > 0 {
		meta.Set(constants.ObjectTypeAuditEvent, ev.Id, "changes", auditFieldChangesFromProto(ev.Changes))
	}

	if ev.MetadataJson != nil && *ev.MetadataJson != "" {
		meta.Set(constants.ObjectTypeAuditEvent, ev.Id, "metadata", rawMessageFromOptionalString(ev.MetadataJson))
	}

	if ev.RequestId != nil && *ev.RequestId != "" {
		meta.Set(constants.ObjectTypeAuditEvent, ev.Id, "request_id", *ev.RequestId)
	}

	return actor
}

func auditFieldChangesFromProto(changes []*pb.AuditFieldChange) *apiresource.List[apiresource.AuditFieldChange] {
	if len(changes) == 0 {
		return nil
	}

	out := make([]apiresource.AuditFieldChange, len(changes))
	for i, c := range changes {
		if c == nil {
			out[i] = apiresource.AuditFieldChange{Object: constants.ObjectTypeAuditFieldChange}
			continue
		}
		out[i] = apiresource.AuditFieldChange{
			Object:   constants.ObjectTypeAuditFieldChange,
			Field:    c.GetField(),
			OldValue: rawMessageFromOptionalString(c.OldValueJson),
			NewValue: rawMessageFromOptionalString(c.NewValueJson),
		}
	}
	return apiresource.NewList(out, apiresource.PageInfo{})
}

func stringPtrFromOptional(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	v := *s
	return &v
}

func rawMessageFromOptionalString(s *string) json.RawMessage {
	if s == nil || *s == "" {
		return nil
	}
	if json.Valid([]byte(*s)) {
		return json.RawMessage(*s)
	}
	encoded, err := json.Marshal(*s)
	if err != nil {
		return nil
	}
	return json.RawMessage(encoded)
}

func stringsFromObjectTypes(ots []constants.ObjectType) []string {
	if len(ots) == 0 {
		return nil
	}
	out := make([]string, len(ots))
	for i, ot := range ots {
		out[i] = string(ot)
	}
	return out
}

func stringsFromActions(actions []constants.AuditAction) []string {
	if len(actions) == 0 {
		return nil
	}
	out := make([]string, len(actions))
	for i, a := range actions {
		out[i] = string(a)
	}
	return out
}
