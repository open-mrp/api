package resourceregistry

import (
	"context"
	"encoding/json"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeAuditEvent,
		Load:       resourceloaders.LoadAuditEvents,
		Subs: []resourcekit.SubField{
			{Key: "actor", Populate: populateActorOnAuditEvent},
			{Key: "changes", Populate: populateChangesOnAuditEvent},
			{Key: "metadata", Populate: populateMetadataOnAuditEvent},
			{
				Key:         "request",
				Target:      constants.ObjectTypeRequestLog,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractRequestIDFromAuditEvent,
				Populate:    populateRequestOnAuditEvent,
			},
		},
	})
}

func populateActorOnAuditEvent(ctx context.Context, parent any, _ map[string]any) {
	ae := parent.(*apiresource.AuditEvent)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAuditEvent, ae.ID, "actor")
	if !ok {
		return
	}
	ae.Actor = v.(*apiresource.Actor)
}

func populateChangesOnAuditEvent(ctx context.Context, parent any, _ map[string]any) {
	ae := parent.(*apiresource.AuditEvent)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAuditEvent, ae.ID, "changes")
	if !ok {
		return
	}
	ae.Changes = v.(*apiresource.List[apiresource.AuditFieldChange])
}

func populateMetadataOnAuditEvent(ctx context.Context, parent any, _ map[string]any) {
	ae := parent.(*apiresource.AuditEvent)
	v, ok := resourcekit.GetLoadMeta(ctx).
		Get(constants.ObjectTypeAuditEvent, ae.ID, "metadata")
	if !ok {
		return
	}
	ae.Metadata = v.(json.RawMessage)
}

func extractRequestIDFromAuditEvent(ctx context.Context, parent any) []string {
	ae := parent.(*apiresource.AuditEvent)
	id, ok := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAuditEvent, ae.ID, "request_id")
	if !ok || id == "" {
		return nil
	}
	return []string{id}
}

func populateRequestOnAuditEvent(ctx context.Context, parent any, loaded map[string]any) {
	ae := parent.(*apiresource.AuditEvent)
	id, ok := resourcekit.GetLoadMeta(ctx).
		GetString(constants.ObjectTypeAuditEvent, ae.ID, "request_id")
	if !ok || id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		ae.Request = v.(*apiresource.RequestLog)
	}
}
