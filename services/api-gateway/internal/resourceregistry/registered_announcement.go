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
		ObjectType: constants.ObjectTypeAnnouncement,
		Load:       resourceloaders.LoadAnnouncements,
		Subs: []resourcekit.SubField{
			// resource is the announcement's link target (a polymorphic Entity). It is built inline by the announcement service and stashed in LoadMeta; Populate gates it on ?include=resource.
			{Key: "resource", Cardinality: resourcekit.CardinalityOnePtr, Populate: populateResourceOnAnnouncement},
		},
	})
}

func populateResourceOnAnnouncement(ctx context.Context, parent any, _ map[string]any) {
	a := parent.(*apiresource.Announcement)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeAnnouncement, a.ID, "resource")
	if !ok {
		return
	}
	a.Resource = v.(*apiresource.Entity)
}
