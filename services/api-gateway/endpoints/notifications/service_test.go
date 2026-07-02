package notificationep

import (
	"context"
	"testing"
	"time"

	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
	pb "github.com/augno/api/shared/proto/notification"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// stashedResource returns the notification's expandable `resource` entity stashed in the request
// LoadMeta (the base resource leaves the field nil; it is surfaced only on ?include=resource).
func stashedResource(ctx context.Context, id string) *apiresource.Entity {
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeNotification, id, "resource")
	if !ok {
		return nil
	}
	return v.(*apiresource.Entity)
}

// stashedSender returns the notification's expandable `sender` actor stashed in the request LoadMeta
// (the base resource leaves the field nil; it is surfaced only on ?include=sender).
func stashedSender(ctx context.Context, id string) *apiresource.Actor {
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeNotification, id, "sender")
	if !ok {
		return nil
	}
	return v.(*apiresource.Actor)
}

func strptr(s string) *string { return &s }

func TestNotificationStatus_Precedence(t *testing.T) {
	now := timestamppb.Now()

	cases := []struct {
		name string
		info *pb.NotificationInfo
		want constants.NotificationStatus
	}{
		{"unseen", &pb.NotificationInfo{}, constants.NotificationStatusUnseen},
		{"seen", &pb.NotificationInfo{SeenAt: now}, constants.NotificationStatusSeen},
		{"read", &pb.NotificationInfo{SeenAt: now, ReadAt: now}, constants.NotificationStatusRead},
		{"read-without-seen", &pb.NotificationInfo{ReadAt: now}, constants.NotificationStatusRead},
		{"dismissed-wins", &pb.NotificationInfo{SeenAt: now, ReadAt: now, DismissedAt: now}, constants.NotificationStatusDismissed},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, notificationStatus(c.info))
		})
	}
}

func TestNotificationFromProto_FullMapping(t *testing.T) {
	created := time.Date(2026, 6, 20, 8, 0, 0, 0, time.UTC)
	updated := created.Add(time.Minute)
	seen := created.Add(2 * time.Minute)

	info := &pb.NotificationInfo{
		Id:               "nf_abc",
		AccountId:        "ac_1",
		Category:         "order.updated",
		Title:            "Order updated",
		Body:             strptr("body text"),
		LinkResourceType: strptr("sales_order"),
		LinkResourceId:   strptr("so_1024"),
		Priority:         "high",
		SeenAt:           timestamppb.New(seen),
		CreatedAt:        timestamppb.New(created),
		UpdatedAt:        timestamppb.New(updated),
	}

	ctx := resourcekit.WithLoadMeta(context.Background())
	n := notificationFromProto(ctx, info)

	assert.Equal(t, "nf_abc", n.ID)
	assert.Equal(t, constants.ObjectTypeNotification, n.Object)
	assert.Equal(t, constants.NotificationCategory("order.updated"), n.Category)
	assert.Equal(t, constants.NotificationStatusSeen, n.Status)
	assert.Equal(t, constants.NotificationPriority("high"), n.Priority)
	assert.Equal(t, "Order updated", n.Title)
	require.NotNil(t, n.Body)
	assert.Equal(t, "body text", *n.Body)
	require.NotNil(t, n.SeenAt)
	assert.True(t, seen.Equal(*n.SeenAt))
	assert.Nil(t, n.ReadAt)
	assert.Nil(t, n.DismissedAt)
	assert.True(t, created.Equal(n.CreatedAt))
	assert.True(t, updated.Equal(n.UpdatedAt))

	// The link is expandable: nil on the base resource, stashed for ?include=resource as a
	// polymorphic Entity reference.
	assert.Nil(t, n.Resource, "resource is expandable and must be nil on the base resource")
	res := stashedResource(ctx, "nf_abc")
	require.NotNil(t, res)
	assert.Equal(t, constants.ObjectTypeEntity, res.Object)
	assert.Equal(t, constants.ObjectType("sales_order"), res.Type)
	assert.Equal(t, "so_1024", res.ID)
}

func TestSenderFromProto(t *testing.T) {
	t.Run("user sender", func(t *testing.T) {
		s := senderFromProto(strptr("user"), strptr("acus_1"), strptr("Jie Yan"))
		require.NotNil(t, s)
		assert.Equal(t, constants.ObjectTypeActor, s.Object)
		assert.Equal(t, constants.ActorTypeUser, s.Type)
		assert.Equal(t, "acus_1", s.ID)
		require.NotNil(t, s.Name)
		assert.Equal(t, "Jie Yan", *s.Name)
	})
	t.Run("group sender maps to group actor", func(t *testing.T) {
		s := senderFromProto(strptr("group"), strptr("sd_1"), strptr("Customer Service"))
		require.NotNil(t, s)
		assert.Equal(t, constants.ActorTypeGroup, s.Type)
	})
	t.Run("agent sender without name", func(t *testing.T) {
		s := senderFromProto(strptr("agent"), strptr("agtc_1"), nil)
		require.NotNil(t, s)
		assert.Equal(t, constants.ActorTypeAgent, s.Type)
		assert.Nil(t, s.Name)
	})
	t.Run("system type → nil sender", func(t *testing.T) {
		assert.Nil(t, senderFromProto(strptr("system"), strptr("anything"), nil))
	})
	t.Run("absent type → nil sender", func(t *testing.T) {
		assert.Nil(t, senderFromProto(nil, nil, nil))
		assert.Nil(t, senderFromProto(strptr(""), nil, nil))
	})
	t.Run("type without id → nil sender", func(t *testing.T) {
		assert.Nil(t, senderFromProto(strptr("user"), nil, strptr("name")))
		assert.Nil(t, senderFromProto(strptr("user"), strptr(""), nil))
	})
}

func TestNotificationFromProto_Sender(t *testing.T) {
	info := &pb.NotificationInfo{
		Id: "nf_s", Category: "agent.run_completed", Title: "Run done", Priority: "normal",
		SenderType: strptr("agent"), SenderId: strptr("agtc_9"), SenderName: strptr("Forecaster"),
		CreatedAt: timestamppb.Now(), UpdatedAt: timestamppb.Now(),
	}
	ctx := resourcekit.WithLoadMeta(context.Background())
	n := notificationFromProto(ctx, info)

	// The sender is expandable: nil on the base resource, stashed for ?include=sender as an Actor.
	assert.Nil(t, n.Sender, "sender is expandable and must be nil on the base resource")
	s := stashedSender(ctx, "nf_s")
	require.NotNil(t, s)
	assert.Equal(t, constants.ActorTypeAgent, s.Type)
	assert.Equal(t, "agtc_9", s.ID)
}

func TestNotificationFromProto_NilsAndEmpties(t *testing.T) {
	info := &pb.NotificationInfo{
		Id:        "nf_x",
		Category:  "system.broadcast",
		Title:     "t",
		Priority:  "normal",
		CreatedAt: timestamppb.Now(),
		UpdatedAt: timestamppb.Now(),
		// no body, link, or status timestamps
	}
	ctx := resourcekit.WithLoadMeta(context.Background())
	n := notificationFromProto(ctx, info)

	assert.Equal(t, constants.NotificationStatusUnseen, n.Status)
	assert.Nil(t, n.Body)
	assert.Nil(t, n.Resource)
	assert.Nil(t, stashedResource(ctx, "nf_x"), "no link → nothing stashed (not a partial Entity)")
	assert.Nil(t, n.SeenAt)
}

func TestNotificationFromProto_PartialLinkIgnored(t *testing.T) {
	// Only a type without an id (or vice versa) is not a valid reference.
	info := &pb.NotificationInfo{
		Id:               "nf_x",
		Category:         "order.updated",
		Title:            "t",
		Priority:         "normal",
		LinkResourceType: strptr("sales_order"),
		CreatedAt:        timestamppb.Now(),
		UpdatedAt:        timestamppb.Now(),
	}
	ctx := resourcekit.WithLoadMeta(context.Background())
	n := notificationFromProto(ctx, info)
	assert.Nil(t, n.Resource)
	assert.Nil(t, stashedResource(ctx, "nf_x"), "a link type without an id must not produce a partial Entity")
}

func TestNotificationFromProto_Nil(t *testing.T) {
	n := notificationFromProto(resourcekit.WithLoadMeta(context.Background()), nil)
	assert.Equal(t, apiresource.Notification{}, n, "nil proto → zero-value resource")
}
