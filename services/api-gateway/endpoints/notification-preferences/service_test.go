package notificationpreferenceep

import (
	"testing"

	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/shared/constants"
	pb "github.com/open-mrp/api/shared/proto/notification"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNotificationPreferenceFromProto_Category(t *testing.T) {
	now := timestamppb.Now()
	p := notificationPreferenceFromProto(&pb.NotificationPreferenceInfo{
		Id: "nfpf_1", AccountId: "ac_1", AccountUserId: "acus_1", Category: "chat.message",
		InAppEnabled: true, EmailEnabled: false, PushEnabled: false, Digest: "instant",
		CreatedAt: now, UpdatedAt: now,
	})
	assert.Equal(t, "nfpf_1", p.ID)
	assert.Equal(t, constants.ObjectTypeNotificationPreference, p.Object)
	require.NotNil(t, p.Category)
	assert.Equal(t, constants.NotificationCategoryChatMessage, *p.Category)
	assert.True(t, p.InAppEnabled)
	assert.False(t, p.EmailEnabled)
	assert.Equal(t, constants.NotificationDigestInstant, p.Digest)
}

func TestNotificationPreferenceFromProto_GlobalDefaultNullCategory(t *testing.T) {
	now := timestamppb.Now()
	p := notificationPreferenceFromProto(&pb.NotificationPreferenceInfo{
		Id: "nfpf_2", AccountId: "ac_1", AccountUserId: "acus_1", Category: "",
		InAppEnabled: true, EmailEnabled: true, Digest: "daily", CreatedAt: now, UpdatedAt: now,
	})
	assert.Nil(t, p.Category, "empty category surfaces as the null global default")
	assert.Equal(t, constants.NotificationDigestDaily, p.Digest)
}

func TestNotificationPreferenceFromProto_Nil(t *testing.T) {
	assert.Equal(t, apiresource.NotificationPreference{}, notificationPreferenceFromProto(nil))
}
