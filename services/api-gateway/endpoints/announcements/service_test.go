package announcementep

import (
	"context"
	"testing"

	"github.com/open-mrp/apikit/include"

	"github.com/open-mrp/api/services/api-gateway/internal/chatmap"
	"github.com/open-mrp/api/services/api-gateway/internal/recordstub"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/notification"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func strptr(s string) *string { return &s }

func TestAnnouncementFromProto(t *testing.T) {
	now := timestamppb.Now()
	info := &pb.AnnouncementInfo{
		Id: "an_1", Scope: "account", Category: "system.broadcast", Title: "Maintenance",
		Body: strptr("tonight"), Priority: "high",
		LinkResourceType: strptr("sales_order"), LinkResourceId: strptr("so_1"),
		PublishAt: now, CreatedAt: now, UpdatedAt: now,
		SeenAt: now, // seen but not read → status seen
	}
	ctx := include.WithLoadMeta(context.Background())
	a := announcementFromProto(ctx, info)
	assert.Equal(t, "an_1", a.ID)
	assert.Equal(t, constants.ObjectTypeAnnouncement, a.Object)
	assert.Equal(t, constants.AnnouncementScopeAccount, a.Scope)
	assert.Equal(t, constants.NotificationStatusSeen, a.Status)
	assert.Equal(t, constants.NotificationPriority("high"), a.Priority)
	// related.record is expandable: nil on the base resource, stashed for ?include=related.record.
	require.NotNil(t, a.Related)
	assert.Nil(t, a.Related.Record)
	v, ok := include.GetLoadMeta(ctx).Get(constants.ObjectTypeAnnouncement, "an_1", chatmap.MetaRelatedRecord)
	require.True(t, ok)
	assert.Equal(t, recordstub.Ref{Type: constants.RecordTypeSalesOrder, ID: "so_1"}, v)
	require.NotNil(t, a.SeenAt)
	assert.Nil(t, a.ReadAt)
}

func TestAnnouncementStatus_Precedence(t *testing.T) {
	now := timestamppb.Now()
	assert.Equal(t, constants.NotificationStatusUnseen, announcementStatus(&pb.AnnouncementInfo{}))
	assert.Equal(t, constants.NotificationStatusSeen, announcementStatus(&pb.AnnouncementInfo{SeenAt: now}))
	assert.Equal(t, constants.NotificationStatusRead, announcementStatus(&pb.AnnouncementInfo{ReadAt: now}))
	assert.Equal(t, constants.NotificationStatusDismissed, announcementStatus(&pb.AnnouncementInfo{ReadAt: now, DismissedAt: now}))
}

func TestAnnouncementFromProto_Nil(t *testing.T) {
	assert.Equal(t, apiresource.Announcement{}, announcementFromProto(include.WithLoadMeta(context.Background()), nil))
}
