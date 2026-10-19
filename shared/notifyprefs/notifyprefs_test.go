package notifyprefs

import (
	"testing"

	"github.com/open-mrp/api/shared/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	inApp = Setting{Channels: []constants.NotificationChannel{constants.NotificationChannelInApp}, Digest: constants.NotificationDigestInstant}
	email = Setting{Channels: []constants.NotificationChannel{constants.NotificationChannelEmail}, Digest: constants.NotificationDigestDaily}
	off   = Setting{Channels: []constants.NotificationChannel{}, Digest: constants.NotificationDigestInstant}
)

func TestResolve_FirstMatchWins(t *testing.T) {
	order := string(constants.NotificationCategoryOrderUpdated)
	cases := []struct {
		name   string
		layers Layers
		want   Setting
	}{
		{"built-in", Layers{}, BuiltIn()},
		{"account default", Layers{Account: Defaults{Default: &email}}, email},
		{"account type over account default", Layers{Account: Defaults{Default: &email, Types: map[string]Setting{order: off}}}, off},
		{"role default over account type", Layers{Role: Defaults{Default: &email}, Account: Defaults{Types: map[string]Setting{order: off}}}, email},
		{"role type over role default", Layers{Role: Defaults{Default: &email, Types: map[string]Setting{order: off}}}, off},
		{"personal default over role type", Layers{Personal: Personal{Default: &inApp}, Role: Defaults{Types: map[string]Setting{order: off}}}, inApp},
		{"personal type over personal default", Layers{Personal: Personal{Default: &inApp, Types: map[string]Setting{order: email}}}, email},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.layers.Resolve(constants.NotificationCategoryOrderUpdated))
		})
	}
}

func TestResolve_RequiredTypeAlwaysEmails(t *testing.T) {
	l := Layers{Personal: Personal{Default: &off}}
	got := l.Resolve(constants.NotificationCategorySecuritySignIn)
	assert.True(t, got.Has(constants.NotificationChannelEmail))
	assert.Empty(t, off.Channels, "the stored setting is not changed")
}

func TestAutoSubscribe_InheritsAndDefaults(t *testing.T) {
	assert.Equal(t, constants.DefaultAutoSubscribe(), Layers{}.AutoSubscribe())
	edited := []constants.AutoSubscribeTrigger{constants.AutoSubscribeEdited}
	assert.Equal(t, edited, Layers{Account: Defaults{AutoSubscribe: edited}}.AutoSubscribe())
	none := []constants.AutoSubscribeTrigger{}
	l := Layers{Personal: Personal{AutoSubscribe: none}, Account: Defaults{AutoSubscribe: edited}}
	assert.Equal(t, none, l.AutoSubscribe())
	assert.False(t, l.AutoSubscribes(constants.SubscriptionReasonCreator))
	assert.True(t, l.AutoSubscribes(constants.SubscriptionReasonManual))
	assert.True(t, Layers{}.AutoSubscribes(constants.SubscriptionReasonCreator))
	assert.False(t, Layers{}.AutoSubscribes(constants.SubscriptionReasonEditor))
}

func TestApply_MergesAndRemoves(t *testing.T) {
	order := string(constants.NotificationCategoryOrderUpdated)
	chat := string(constants.NotificationCategoryChatMessage)
	d := Defaults{Default: &email, Types: map[string]Setting{order: off, chat: inApp}}
	auto := []constants.AutoSubscribeTrigger{constants.AutoSubscribeCreated, constants.AutoSubscribeCreated}
	got := d.Apply(Patch{
		Default:       &SettingPatch{},
		Types:         map[string]*Setting{order: nil, chat: {Channels: []constants.NotificationChannel{constants.NotificationChannelPush}}},
		AutoSubscribe: &AutoSubscribePatch{Value: &auto},
	})
	assert.Nil(t, got.Default)
	assert.NotContains(t, got.Types, order)
	assert.Equal(t, Setting{Channels: []constants.NotificationChannel{constants.NotificationChannelPush}, Digest: constants.NotificationDigestInstant}, got.Types[chat])
	assert.Equal(t, []constants.AutoSubscribeTrigger{constants.AutoSubscribeCreated}, got.AutoSubscribe)
	assert.Len(t, d.Types, 2, "the original is not changed")
}

func TestValidate(t *testing.T) {
	required := string(constants.NotificationCategorySecurityDevicePaired)
	err := Patch{Types: map[string]*Setting{required: &off}}.Validate("")
	require.NotNil(t, err)
	assert.Equal(t, "types."+required+".channels", err.Errors[0].Param)

	err = Patch{Types: map[string]*Setting{"nope": &off}}.Validate("notification_defaults")
	require.NotNil(t, err)
	assert.Equal(t, "notification_defaults.types.nope", err.Errors[0].Param)

	bad := Setting{Channels: []constants.NotificationChannel{"sms"}}
	require.NotNil(t, Patch{Default: &SettingPatch{Value: &bad}}.Validate(""))
	assert.Nil(t, Patch{Default: &SettingPatch{Value: &email}, Types: map[string]*Setting{required: nil}}.Validate(""))
}

func TestParseMarshalRoundTrip(t *testing.T) {
	d := Defaults{Default: &email, Types: map[string]Setting{"chat.message": off}, AutoSubscribe: []constants.AutoSubscribeTrigger{}}
	assert.Equal(t, d, Parse(d.Marshal()))
	assert.Equal(t, Empty(), Parse(nil))
	assert.Equal(t, Empty(), Parse([]byte("not json")))
}
