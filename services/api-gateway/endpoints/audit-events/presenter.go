package auditeventsep

import (
	"encoding/json"
	"time"

	grpcutil "github.com/augno/api/services/api-gateway/internal/grpc"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/platform"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func AuditEventPresenter(ev *pb.AuditEventInfo) *apiresource.AuditEvent {
	if ev == nil {
		return &apiresource.AuditEvent{}
	}

	result := &apiresource.AuditEvent{
		ID:           ev.Id,
		Object:       constants.ObjectTypeAuditEvent,
		Action:       constants.AuditAction(ev.Action),
		ResourceType: constants.ObjectType(ev.ResourceType),
		ResourceID:   ev.ResourceId,
		Metadata:     rawMessageFromOptionalString(ev.MetadataJson),
		OccurredAt:   timestampToTime(ev.OccurredAt),
		CreatedAt:    timestampToTime(ev.CreatedAt),
	}

	if ev.Actor != nil {
		var name *string
		if ev.Actor.Name != nil && *ev.Actor.Name != "" {
			n := *ev.Actor.Name
			name = &n
		}
		result.Actor = apiresource.NewActor(
			ev.Actor.Id,
			constants.ActorType(ev.Actor.ActorType),
			name,
			stringPtrFromOptional(ev.Actor.Handle),
		)
	}

	result.Changes = AuditFieldChangesPresenter(ev.Changes)

	result.RequestID = stringPtrFromOptional(ev.RequestId)
	result.IdempotencyKeyID = stringPtrFromOptional(ev.IdempotencyKeyId)
	result.SourceIP = stringPtrFromOptional(ev.SourceIp)

	return result
}

func AuditEventListPresenter(resp *pb.ListAuditEventsResponse) *apiresource.List[apiresource.AuditEvent] {
	if resp == nil {
		return apiresource.NewList[apiresource.AuditEvent](nil, grpcutil.MapProtoPageInfo(nil))
	}

	events := make([]apiresource.AuditEvent, len(resp.AuditEvents))
	for i, ev := range resp.AuditEvents {
		presented := AuditEventPresenter(ev)
		if presented != nil {
			events[i] = *presented
		}
	}

	return apiresource.NewList(events, grpcutil.MapProtoPageInfo(resp.PageInfo))
}

func AuditFieldChangesPresenter(changes []*pb.AuditFieldChange) *apiresource.List[apiresource.AuditFieldChange] {
	if len(changes) == 0 {
		return nil
	}

	out := make([]apiresource.AuditFieldChange, len(changes))
	for i, c := range changes {
		if c == nil {
			out[i] = apiresource.AuditFieldChange{}
			continue
		}
		out[i] = apiresource.AuditFieldChange{
			Field:    c.GetField(),
			OldValue: rawMessageFromOptionalString(c.OldValueJson),
			NewValue: rawMessageFromOptionalString(c.NewValueJson),
		}
	}
	return apiresource.NewList(out, apiresource.PageInfo{})
}

func timestampToTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
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
	return json.RawMessage(*s)
}
