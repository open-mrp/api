// Package notifyprefs holds the notification settings shape shared by a person's preferences, a role's defaults and
// an account's defaults, and the order in which they resolve.
package notifyprefs

import (
	"encoding/json"
	"slices"

	"github.com/open-mrp/api/shared/constants"
)

// Setting is where one notification type goes.
type Setting struct {
	Channels []constants.NotificationChannel `json:"channels"`
	Digest   constants.NotificationDigest    `json:"digest"`
}

// Has reports whether the setting delivers on channel.
func (s Setting) Has(channel constants.NotificationChannel) bool {
	return slices.Contains(s.Channels, channel)
}

// Defaults is one level of settings (a role's or the account's). A nil Default or AutoSubscribe inherits from the
// next level.
type Defaults struct {
	Default       *Setting                         `json:"default"`
	Types         map[string]Setting               `json:"types"`
	AutoSubscribe []constants.AutoSubscribeTrigger `json:"auto_subscribe"`
}

// Empty is a level that sets nothing.
func Empty() Defaults {
	return Defaults{Types: map[string]Setting{}}
}

// Parse reads stored defaults; empty input or invalid JSON is a level that sets nothing.
func Parse(raw []byte) Defaults {
	d := Empty()
	if len(raw) == 0 {
		return d
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		return Empty()
	}
	if d.Types == nil {
		d.Types = map[string]Setting{}
	}
	return d
}

// Marshal is the stored form of d.
func (d Defaults) Marshal() []byte {
	if d.Types == nil {
		d.Types = map[string]Setting{}
	}
	b, _ := json.Marshal(d)
	return b
}

// BuiltIn applies when nothing else is set: in-app on, email and push off.
func BuiltIn() Setting {
	return Setting{Channels: []constants.NotificationChannel{constants.NotificationChannelInApp}, Digest: constants.NotificationDigestInstant}
}

// Personal is a person's own settings: per-type overrides, an optional default and an optional auto-subscribe list.
type Personal struct {
	Default       *Setting
	Types         map[string]Setting
	AutoSubscribe []constants.AutoSubscribeTrigger
}

// Layers is everything that decides where one person's notifications go.
type Layers struct {
	Personal Personal
	// Role is the person's role defaults.
	Role    Defaults
	Account Defaults
}

// Resolve picks the setting for a notification type, first match wins: personal type, personal default, role type,
// role default, account type, account default, built-in. A required type always emails.
func (l Layers) Resolve(t constants.NotificationCategory) Setting {
	s := l.resolve(string(t))
	if t.Required() && !s.Has(constants.NotificationChannelEmail) {
		s.Channels = append(slices.Clone(s.Channels), constants.NotificationChannelEmail)
	}
	return s
}

func (l Layers) resolve(t string) Setting {
	if s, ok := l.Personal.Types[t]; ok {
		return s
	}
	if l.Personal.Default != nil {
		return *l.Personal.Default
	}
	if s, ok := l.Role.Types[t]; ok {
		return s
	}
	if l.Role.Default != nil {
		return *l.Role.Default
	}
	if s, ok := l.Account.Types[t]; ok {
		return s
	}
	if l.Account.Default != nil {
		return *l.Account.Default
	}
	return BuiltIn()
}

// EffectiveDefault is the person's default: their own, else their role's, else the account's, else the built-in.
func (l Layers) EffectiveDefault() Setting {
	if l.Personal.Default != nil {
		return *l.Personal.Default
	}
	if l.Role.Default != nil {
		return *l.Role.Default
	}
	if l.Account.Default != nil {
		return *l.Account.Default
	}
	return BuiltIn()
}

// AutoSubscribe is the person's auto-subscribe list: theirs, else their role's, else the account's, else the built-in.
func (l Layers) AutoSubscribe() []constants.AutoSubscribeTrigger {
	if l.Personal.AutoSubscribe != nil {
		return l.Personal.AutoSubscribe
	}
	if l.Role.AutoSubscribe != nil {
		return l.Role.AutoSubscribe
	}
	if l.Account.AutoSubscribe != nil {
		return l.Account.AutoSubscribe
	}
	return constants.DefaultAutoSubscribe()
}

// AutoSubscribes reports whether the person is subscribed automatically for this reason.
func (l Layers) AutoSubscribes(reason constants.SubscriptionReason) bool {
	if reason == constants.SubscriptionReasonManual {
		return true
	}
	return slices.Contains(l.AutoSubscribe(), reason.AutoSubscribeTrigger())
}

// TemplateDefaults are the notification defaults a role starts with when created from a template. No template sets
// any yet: the only per-role example (shipment emails for warehouse) has no notification type to point at.
func TemplateDefaults(constants.RoleTemplate) Defaults {
	return Empty()
}
