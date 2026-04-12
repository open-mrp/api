package grpc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	pb "github.com/augno/api/shared/proto/platform"
	"github.com/augno/api/shared/tracing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var auditGRPCHandlerTracer = tracing.GetTracer("platform-service.audit_grpc_handler")

type auditHandler struct {
	pb.UnimplementedAuditServiceServer
	auditSvc domain.AuditEventSvc
}

func NewAuditHandler(server *grpc.Server, auditSvc domain.AuditEventSvc) *auditHandler {
	handler := &auditHandler{
		auditSvc: auditSvc,
	}
	pb.RegisterAuditServiceServer(server, handler)
	return handler
}

func (h *auditHandler) CreateAuditEvent(ctx context.Context, req *pb.CreateAuditEventRequest) (*pb.CreateAuditEventResponse, error) {
	ctx, span := auditGRPCHandlerTracer.Start(ctx, "grpc_handler.create_audit_event")
	defer span.End()

	if req == nil || req.AuditEvent == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	if req.AuditEvent.Actor == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewValidationError("Audit event actor is required."))
	}

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewInvariantViolationError("Identity not found in context."))
	}
	if identity.Target == nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apierror.NewAuthenticationError("The Augno-Account header is required."))
	}

	info := req.AuditEvent

	// Map actor and changes.
	changes := make([]domain.AuditFieldChange, len(info.Changes))
	for i, c := range info.Changes {
		var oldVal json.RawMessage
		if c.OldValueJson != nil {
			oldVal = json.RawMessage(*c.OldValueJson)
		}
		var newVal json.RawMessage
		if c.NewValueJson != nil {
			newVal = json.RawMessage(*c.NewValueJson)
		}
		changes[i] = domain.AuditFieldChange{
			Field:    c.GetField(),
			OldValue: oldVal,
			NewValue: newVal,
		}
	}

	var metadata json.RawMessage
	if info.MetadataJson != nil && *info.MetadataJson != "" {
		metadata = json.RawMessage(*info.MetadataJson)
	}

	occurredAt := time.Now().UTC()
	if info.OccurredAt != nil {
		occurredAt = info.OccurredAt.AsTime()
	}

	event := &domain.AuditEvent{
		ID:           info.GetId(),
		ActorID:      info.Actor.GetId(),
		ActorType:    info.Actor.GetType(),
		IdentityType: info.Actor.GetIdentityType(),
		AccountID:    identity.Target.AccountID,

		Action:       constants.AuditAction(info.GetAction()),
		ResourceType: constants.ObjectType(info.GetResourceType()),
		ResourceID:   info.GetResourceId(),
		Changes:      changes,
		Metadata:     metadata,

		ServiceName:      info.GetServiceName(),
		RequestID:        info.RequestId,
		IdempotencyKeyID: info.IdempotencyKeyId,
		SourceIP:         info.SourceIp,

		OccurredAt: occurredAt,
	}

	if apiErr := h.auditSvc.SaveAuditEvent(ctx, event); apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.CreateAuditEventResponse{Success: true}, nil
}

func (h *auditHandler) ListAuditEvents(ctx context.Context, req *pb.ListAuditEventsRequest) (*pb.ListAuditEventsResponse, error) {
	ctx, span := auditGRPCHandlerTracer.Start(ctx, "grpc_handler.list_audit_events")
	defer span.End()

	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	filter := &domain.ListAuditEventsFilter{
		ResourceType: req.ResourceType,
		ResourceID:   req.ResourceId,
		ActorID:      req.ActorId,
		Action:       req.Action,
		AccountID:    req.AccountId,
		Query:        req.Query,
		Cursor:       req.Cursor,
		Limit:        req.Limit,
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		filter.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		filter.EndDate = &t
	}

	result, apiErr := h.auditSvc.ListAuditEvents(ctx, filter, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	pbEvents := make([]*pb.AuditEventInfo, len(result.AuditEvents))
	for i, ev := range result.AuditEvents {
		pbEvents[i] = auditEventToProto(ev)
	}

	return &pb.ListAuditEventsResponse{
		AuditEvents: pbEvents,
		PageInfo: &pb.PageInfo{
			NextCursor:  result.PageInfo.NextCursor,
			PrevCursor:  result.PageInfo.PrevCursor,
			HasNextPage: result.PageInfo.HasNextPage,
			HasPrevPage: result.PageInfo.HasPrevPage,
		},
	}, nil
}

func (h *auditHandler) GetAuditEvent(ctx context.Context, req *pb.GetAuditEventRequest) (*pb.GetAuditEventResponse, error) {
	ctx, span := auditGRPCHandlerTracer.Start(ctx, "grpc_handler.get_audit_event")
	defer span.End()

	if req == nil {
		return nil, contracts.NewMissingGRPCRequestDataError()
	}

	ev, apiErr := h.auditSvc.GetAuditEvent(ctx, req.Id, req.Includes)
	if apiErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(apiErr)
	}

	return &pb.GetAuditEventResponse{
		AuditEvent: auditEventToProto(ev),
	}, nil
}

func auditEventToProto(ev *domain.AuditEventRead) *pb.AuditEventInfo {
	if ev == nil {
		return &pb.AuditEventInfo{}
	}

	var metadata *string
	if len(ev.Metadata) > 0 {
		s := string(ev.Metadata)
		metadata = &s
	}

	pbChanges := make([]*pb.AuditFieldChange, len(ev.Changes))
	for i, c := range ev.Changes {
		pbc := &pb.AuditFieldChange{
			Field: c.Field,
		}
		if len(c.OldValue) > 0 {
			s := string(c.OldValue)
			pbc.OldValueJson = &s
		}
		if len(c.NewValue) > 0 {
			s := string(c.NewValue)
			pbc.NewValueJson = &s
		}
		pbChanges[i] = pbc
	}

	var serviceName *string
	if ev.ServiceName != "" {
		s := ev.ServiceName
		serviceName = &s
	}

	var occurredAt *timestamppb.Timestamp
	if !ev.OccurredAt.IsZero() {
		occurredAt = timestamppb.New(ev.OccurredAt)
	}
	var createdAt *timestamppb.Timestamp
	if !ev.CreatedAt.IsZero() {
		createdAt = timestamppb.New(ev.CreatedAt)
	}

	var actor *pb.AuditActor
	if ev.Actor != nil {
		actor = &pb.AuditActor{
			Id:           ev.Actor.ID,
			ActorType:    string(ev.Actor.ActorType),
			Type:         ev.Actor.Type,
			IdentityType: ev.Actor.IdentityType,
			Name:         ev.Actor.Name,
			Handle:       ev.Actor.Handle,
		}
	}

	return &pb.AuditEventInfo{
		Id:               ev.ID,
		Action:           string(ev.Action),
		ResourceType:     string(ev.ResourceType),
		ResourceId:       ev.ResourceID,
		Actor:            actor,
		Changes:          pbChanges,
		MetadataJson:     metadata,
		ServiceName:      serviceName,
		RequestId:        ev.RequestID,
		IdempotencyKeyId: ev.IdempotencyKeyID,
		SourceIp:         ev.SourceIP,
		OccurredAt:       occurredAt,
		CreatedAt:        createdAt,
	}
}
