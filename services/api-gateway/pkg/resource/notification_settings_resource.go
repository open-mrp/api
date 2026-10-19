package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/notifyprefs"
	apierror "github.com/open-mrp/apikit/apierror"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/timeutil"
)

// Where one type of notification goes.
type NotificationSetting struct {
	// The channels the notification is delivered on: `in_app` (your notification feed), `email` and `push`. An empty list turns the notification off.
	//
	// Push notifications aren't sent yet; the choice is kept for when they are.
	Channels []constants.NotificationChannel `json:"channels" validate:"required"`
	// How often email is sent: `instant` (as it happens), `hourly` or `daily`.
	//
	// Hourly and daily digests aren't sent yet, so email set to either is held back for now.
	Digest constants.NotificationDigest `json:"digest" validate:"required"`
}

// Starting notification settings, applied before a person's own preferences.
type NotificationSettings struct {
	// Where notifications go when `types` has no entry for their type. Null leaves it to the next level (the account's defaults, then OpenMRP's: in-app only).
	Default *NotificationSetting `json:"default"`
	// Settings for single notification types, keyed by type such as `order.updated`.
	Types map[string]NotificationSetting `json:"types" validate:"required"`
	// When people are subscribed to a record automatically: `created` (they created it), `assigned` (they were assigned to it, such as its sales rep), `mentioned` (they were @mentioned about it) and `edited` (they edited it). Null leaves it to the next level; OpenMRP's is `created`, `assigned` and `mentioned`.
	AutoSubscribe *[]constants.AutoSubscribeTrigger `json:"auto_subscribe"`
}

// Your team's starting notification settings.
//
// A role's defaults apply before the account's, and each person's own preferences take precedence.
type NotificationDefaults struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=notification_defaults"`
	// Where notifications go when `types` has no entry for their type. Null leaves it to OpenMRP's: in-app only.
	Default *NotificationSetting `json:"default"`
	// Settings for single notification types, keyed by type such as `order.updated`.
	Types map[string]NotificationSetting `json:"types" validate:"required"`
	// When people are subscribed to a record automatically: `created`, `assigned`, `mentioned`, `edited`. Null leaves it to OpenMRP's: `created`, `assigned` and `mentioned`.
	AutoSubscribe *[]constants.AutoSubscribeTrigger `json:"auto_subscribe"`
}

// NotificationDefaultsOf is the response shape of the account's notification defaults.
func NotificationDefaultsOf(d notifyprefs.Defaults) *NotificationDefaults {
	s := NotificationSettingsOf(d)
	return &NotificationDefaults{Object: constants.ObjectTypeNotificationDefaults, Default: s.Default, Types: s.Types, AutoSubscribe: s.AutoSubscribe}
}

// Where your notifications go.
//
// `default` applies to every type you haven't set in `types`. An empty `channels` list turns a type off. Required types can't be turned off.
type NotificationPreferences struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=notification_preferences"`
	// Where notifications go when you haven't set their type in `types`: your own default, else your role's, else the account's, else in-app only.
	Default NotificationSetting `json:"default" validate:"required"`
	// Your settings for single notification types, keyed by type such as `order.updated`.
	Types map[string]NotificationSetting `json:"types" validate:"required"`
	// When you're subscribed to a record automatically: `created`, `assigned`, `mentioned`, `edited`.
	AutoSubscribe []constants.AutoSubscribeTrigger `json:"auto_subscribe" validate:"required"`
	// Holds email and push notifications until this time. In-app notifications still arrive.
	PausedUntil *time.Time `json:"paused_until"`
}

var sampleEmailSetting = NotificationSetting{
	Channels: []constants.NotificationChannel{constants.NotificationChannelInApp, constants.NotificationChannelEmail},
	Digest:   constants.NotificationDigestInstant,
}

var SampleNotificationPreferences = &NotificationPreferences{
	Object: constants.ObjectTypeNotificationPreferences,
	Default: NotificationSetting{
		Channels: []constants.NotificationChannel{constants.NotificationChannelInApp},
		Digest:   constants.NotificationDigestInstant,
	},
	Types:         map[string]NotificationSetting{string(constants.NotificationCategoryOrderUpdated): sampleEmailSetting},
	AutoSubscribe: constants.DefaultAutoSubscribe(),
	PausedUntil:   timeutil.TimestampToTimePtr(sampleExpiresAtTimestamp),
}

func (*NotificationPreferences) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleNotificationPreferences)
}

var SampleNotificationSettings = NotificationSettings{
	Default:       nil,
	Types:         map[string]NotificationSetting{string(constants.NotificationCategoryOrderUpdated): sampleEmailSetting},
	AutoSubscribe: &[]constants.AutoSubscribeTrigger{constants.AutoSubscribeCreated, constants.AutoSubscribeAssigned},
}

var SampleNotificationDefaults = &NotificationDefaults{
	Object:        constants.ObjectTypeNotificationDefaults,
	Default:       SampleNotificationSettings.Default,
	Types:         SampleNotificationSettings.Types,
	AutoSubscribe: SampleNotificationSettings.AutoSubscribe,
}

func (*NotificationDefaults) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleNotificationDefaults)
}

func notificationSettingOf(s notifyprefs.Setting) NotificationSetting {
	channels := s.Channels
	if channels == nil {
		channels = []constants.NotificationChannel{}
	}
	return NotificationSetting{Channels: channels, Digest: s.Digest}
}

// NotificationSettingsOf is the response shape of stored notification settings.
func NotificationSettingsOf(d notifyprefs.Defaults) NotificationSettings {
	out := NotificationSettings{Types: make(map[string]NotificationSetting, len(d.Types))}
	if d.Default != nil {
		s := notificationSettingOf(*d.Default)
		out.Default = &s
	}
	for t, s := range d.Types {
		out.Types[t] = notificationSettingOf(s)
	}
	if d.AutoSubscribe != nil {
		auto := append([]constants.AutoSubscribeTrigger{}, d.AutoSubscribe...)
		out.AutoSubscribe = &auto
	}
	return out
}

// NotificationPreferencesOf is the response shape of a person's effective settings.
func NotificationPreferencesOf(d notifyprefs.Defaults, pausedUntil *time.Time) *NotificationPreferences {
	settings := NotificationSettingsOf(d)
	out := &NotificationPreferences{
		Object:        constants.ObjectTypeNotificationPreferences,
		Default:       notificationSettingOf(notifyprefs.BuiltIn()),
		Types:         settings.Types,
		AutoSubscribe: []constants.AutoSubscribeTrigger{},
		PausedUntil:   pausedUntil,
	}
	if settings.Default != nil {
		out.Default = *settings.Default
	}
	if settings.AutoSubscribe != nil {
		out.AutoSubscribe = *settings.AutoSubscribe
	}
	return out
}

// Where one type of notification goes.
type NotificationSettingInput struct {
	// The channels to deliver on: `in_app`, `email`, `push`. Send an empty list to turn the notification off.
	Channels []constants.NotificationChannel `json:"channels"`
	// How often email is sent: `instant`, `hourly` or `daily`. Defaults to `instant`.
	Digest field.Optional[constants.NotificationDigest] `json:"digest,omitzero"`
}

func (in NotificationSettingInput) setting() notifyprefs.Setting {
	s := notifyprefs.Setting{Channels: in.Channels}
	if d, ok := in.Digest.Value(); ok {
		s.Digest = d
	}
	return s
}

// NotificationSettingsPatch is the request shape that changes one level of notification settings.
type NotificationSettingsPatch struct {
	// Where notifications go when `types` has no entry for their type. Send `null` to remove it.
	Default field.Clearable[NotificationSettingInput] `json:"default,omitzero"`
	// Settings for single notification types, keyed by type such as `order.updated`. Types you leave out are unchanged; send a type as `null` to remove its setting.
	Types map[string]*NotificationSettingInput `json:"types,omitzero"`
	// When people are subscribed to records automatically: `created`, `assigned`, `mentioned`, `edited`. Send `null` to remove the list.
	AutoSubscribe field.Clearable[[]constants.AutoSubscribeTrigger] `json:"auto_subscribe,omitzero"`
}

// ToPatch converts the request into a settings patch, checking it. prefix names the request field that holds it.
func (p NotificationSettingsPatch) ToPatch(prefix string) (notifyprefs.Patch, *apierror.APIError) {
	var out notifyprefs.Patch
	join := func(f string) string {
		if prefix == "" {
			return f
		}
		return prefix + "." + f
	}
	if p.Default.IsClear() {
		out.Default = &notifyprefs.SettingPatch{}
	} else if v, ok := p.Default.Value(); ok {
		if v.Channels == nil {
			return out, apierror.NewMissingFieldError(join("default.channels"), "channels is required.")
		}
		s := v.setting()
		out.Default = &notifyprefs.SettingPatch{Value: &s}
	}
	if len(p.Types) > 0 {
		out.Types = make(map[string]*notifyprefs.Setting, len(p.Types))
		for t, v := range p.Types {
			if v == nil {
				out.Types[t] = nil
				continue
			}
			if v.Channels == nil {
				return out, apierror.NewMissingFieldError(join("types."+t+".channels"), "channels is required.")
			}
			s := v.setting()
			out.Types[t] = &s
		}
	}
	if p.AutoSubscribe.IsClear() {
		out.AutoSubscribe = &notifyprefs.AutoSubscribePatch{}
	} else if v, ok := p.AutoSubscribe.Value(); ok {
		if v == nil {
			v = []constants.AutoSubscribeTrigger{}
		}
		out.AutoSubscribe = &notifyprefs.AutoSubscribePatch{Value: &v}
	}
	if apiErr := out.Validate(prefix); apiErr != nil {
		return out, apiErr
	}
	return out, nil
}

// Settings is the full settings a create request sets.
func (p NotificationSettingsPatch) Settings(prefix string) (notifyprefs.Defaults, *apierror.APIError) {
	patch, apiErr := p.ToPatch(prefix)
	if apiErr != nil {
		return notifyprefs.Defaults{}, apiErr
	}
	return notifyprefs.Empty().Apply(patch), nil
}
