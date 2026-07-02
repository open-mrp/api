package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeMessageAttachment,
		Load:       resourceloaders.LoadMessageAttachments,
		Subs: []resourcekit.SubField{
			// resource is the attachment's linked record (a polymorphic Entity) for `resource`-kind
			// attachments. Built inline by chatmap.AttachmentFromProto and stashed in LoadMeta; Populate
			// gates it on ?include=attachments.resource (the attachment is itself reached by traversal
			// from the parent message's attachments include).
			{Key: "resource", Cardinality: resourcekit.CardinalityOnePtr, Populate: populateResourceOnAttachment},
		},
	})
}

func populateResourceOnAttachment(ctx context.Context, parent any, _ map[string]any) {
	a := parent.(*apiresource.MessageAttachment)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeMessageAttachment, a.ID, "resource")
	if !ok {
		return
	}
	a.Resource = v.(*apiresource.Entity)
}
