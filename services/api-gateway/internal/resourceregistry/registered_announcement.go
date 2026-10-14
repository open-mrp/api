package resourceregistry

import (
	"context"
	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	include.Register(&include.Definition{
		ObjectType: constants.ObjectTypeAnnouncement,
		Load:       resourceloaders.LoadAnnouncements,
		Subs: []include.SubField{
			// resource is the announcement's link target (a polymorphic Entity). It is built inline by the announcement service and stashed in LoadMeta; Populate gates it on ?include=resource.
			{Key: "resource", Cardinality: include.CardinalityOnePtr, Populate: populateResourceOnAnnouncement},
		},
	})
}

func populateResourceOnAnnouncement(ctx context.Context, parent any, _ map[string]any) {
	a := parent.(*apiresource.Announcement)
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeAnnouncement, a.ID, "resource")
	if !ok {
		return
	}
	a.Resource = v.(*apiresource.Entity)
}
