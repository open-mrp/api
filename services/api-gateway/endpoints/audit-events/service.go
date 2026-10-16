package auditeventsep

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/domain"
	grpcutil "github.com/open-mrp/api/services/api-gateway/internal/grpc"
	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/platform"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
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
		ResourceTypes:     stringsFromObjectTypes(req.ResourceTypes),
		ResourceIds:       req.ResourceIDs,
		ActorIds:          req.ActorIDs,
		ActorTypes:        actorTypes,
		Actions:           stringsFromActions(req.Actions),
		ActorAccountIds:   req.ActorAccountIDs,
		TargetAccountIds:  req.TargetAccountIDs,
		Query:             req.Query,
		Cursor:            req.Cursor,
		Limit:             req.Limit,
		RootResourceTypes: stringsFromObjectTypes(req.RootResourceTypes),
		RootResourceIds:   req.RootResourceIDs,
		Includes:          include.FilterIncludes(ctx, "changes", "metadata"),
	}
	// Both are needed: the root key leads with the type, so IDs alone would walk every event of the account.
	if len(req.RootResourceIDs) > 0 && len(req.RootResourceTypes) == 0 {
		return nil, apierror.NewParameterMissingError("root_resource_types", "root_resource_types is required with root_resource_ids.")
	}
	if len(req.RootResourceTypes) > 0 && len(req.RootResourceIDs) == 0 {
		return nil, apierror.NewParameterMissingError("root_resource_ids", "root_resource_ids is required with root_resource_types.")
	}

	if req.OccurredAfter != nil && !req.OccurredAfter.IsZero() {
		pbReq.StartDate = timestamppb.New(*req.OccurredAfter)
	}
	if req.OccurredBefore != nil && !req.OccurredBefore.IsZero() {
		pbReq.EndDate = timestamppb.New(*req.OccurredBefore)
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

	meta := include.GetLoadMeta(ctx)
	events := make([]apiresource.AuditEvent, len(resp.AuditEvents))
	actors := make([]*apiresource.ActorRef, 0, len(resp.AuditEvents))
	for i, ev := range resp.AuditEvents {
		events[i] = auditEventFromProto(ev)
		stashAuditEventMeta(meta, ev)
		actors = append(actors, events[i].Actor)
	}
	resourceloaders.ResolveActorRefs(ctx, actors)

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
		Includes: include.FilterIncludes(ctx, "changes", "metadata"),
	}

	resp, apiErr := grpcutil.CallRPC(ctx, auditEventSvcTracer, "service.audit_events.get", domain.ServiceName,
		func(ctx context.Context, opts ...grpc.CallOption) (*pb.GetAuditEventResponse, error) {
			return m.auditClient.GetAuditEvent(ctx, pbReq, opts...)
		})

	if apiErr != nil {
		return nil, apiErr
	}

	result := auditEventFromProto(resp.AuditEvent)
	stashAuditEventMeta(include.GetLoadMeta(ctx), resp.AuditEvent)
	resourceloaders.ResolveActorRefs(ctx, []*apiresource.ActorRef{result.Actor})
	return &result, nil
}

func auditEventFromProto(ev *pb.AuditEventInfo) apiresource.AuditEvent {
	if ev == nil {
		return apiresource.AuditEvent{}
	}

	return apiresource.AuditEvent{
		Actor:          auditActorRef(ev.Actor),
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

// auditActorRef is the event's actor. platform-service joins user and API key names; ResolveActorRefs fills the rest.
func auditActorRef(a *pb.AuditActor) *apiresource.ActorRef {
	if a == nil {
		return nil
	}
	return resourceloaders.ActorRefFromRecorded(constants.ActorType(a.ActorType), a.Id, a.Name)
}

// stashAuditEventMeta stashes the event's expandable sub-resources into the load meta.
func stashAuditEventMeta(meta *include.LoadMeta, ev *pb.AuditEventInfo) {
	if ev == nil {
		return
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
