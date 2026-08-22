package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeNotification,
		Load:       resourceloaders.LoadNotifications,
		Subs: []resourcekit.SubField{
			// sender and resource are built by the notification service and stashed in LoadMeta; Populate gates each on ?include=sender / ?include=resource.
			{Key: "sender", Cardinality: resourcekit.CardinalityOnePtr, Populate: populateSenderOnNotification},
			{Key: "resource", Cardinality: resourcekit.CardinalityOnePtr, Populate: populateResourceOnNotification},
		},
	})
}

func populateSenderOnNotification(ctx context.Context, parent any, _ map[string]any) {
	n := parent.(*apiresource.Notification)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeNotification, n.ID, "sender")
	if !ok {
		return
	}
	n.Sender = v.(*apiresource.Actor)
}

func populateResourceOnNotification(ctx context.Context, parent any, _ map[string]any) {
	n := parent.(*apiresource.Notification)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeNotification, n.ID, "resource")
	if !ok {
		return
	}
	n.Resource = v.(*apiresource.Entity)
}
