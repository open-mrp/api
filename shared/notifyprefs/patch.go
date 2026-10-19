package notifyprefs

import (
	"fmt"
	"maps"

	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
)

// Patch changes one level of settings. Absent parts are left alone. A present Default or AutoSubscribe with a nil
// value removes it (the level inherits again); a nil entry in Types removes that type's override.
type Patch struct {
	Default       *SettingPatch       `json:"default,omitempty"`
	Types         map[string]*Setting `json:"types,omitempty"`
	AutoSubscribe *AutoSubscribePatch `json:"auto_subscribe,omitempty"`
}

// SettingPatch sets a setting, or removes it when Value is nil.
type SettingPatch struct {
	Value *Setting `json:"value"`
}

// AutoSubscribePatch sets the auto-subscribe list, or removes it when Value is nil.
type AutoSubscribePatch struct {
	Value *[]constants.AutoSubscribeTrigger `json:"value"`
}

// Apply returns d with p applied. Validate p first.
func (d Defaults) Apply(p Patch) Defaults {
	out := Defaults{Default: d.Default, Types: maps.Clone(d.Types), AutoSubscribe: d.AutoSubscribe}
	if out.Types == nil {
		out.Types = map[string]Setting{}
	}
	if p.Default != nil {
		if p.Default.Value == nil {
			out.Default = nil
		} else {
			s := normalized(*p.Default.Value)
			out.Default = &s
		}
	}
	for t, s := range p.Types {
		if s == nil {
			delete(out.Types, t)
			continue
		}
		out.Types[t] = normalized(*s)
	}
	if p.AutoSubscribe != nil {
		if p.AutoSubscribe.Value == nil {
			out.AutoSubscribe = nil
		} else {
			out.AutoSubscribe = dedupe(*p.AutoSubscribe.Value)
			if out.AutoSubscribe == nil {
				out.AutoSubscribe = []constants.AutoSubscribeTrigger{}
			}
		}
	}
	return out
}

// normalized fills the default digest and drops repeated channels.
func normalized(s Setting) Setting {
	if s.Digest == "" {
		s.Digest = constants.NotificationDigestInstant
	}
	s.Channels = dedupe(s.Channels)
	if s.Channels == nil {
		s.Channels = []constants.NotificationChannel{}
	}
	return s
}

func dedupe[T comparable](in []T) []T {
	var out []T
	seen := map[T]bool{}
	for _, v := range in {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// Validate checks p's values. prefix names the request field that holds the settings (empty at the top level).
func (p Patch) Validate(prefix string) *apierror.APIError {
	var errs []apierror.FieldError
	if p.Default != nil && p.Default.Value != nil {
		errs = append(errs, validateSetting(join(prefix, "default"), *p.Default.Value, "")...)
	}
	for t, s := range p.Types {
		param := join(prefix, "types."+t)
		if !constants.NotificationCategory(t).IsValid() {
			errs = append(errs, apierror.FieldError{Param: param, Code: apierror.CodeInvalidFormat, Message: fmt.Sprintf("%q isn't a notification type.", t)})
			continue
		}
		if s == nil {
			continue
		}
		errs = append(errs, validateSetting(param, *s, constants.NotificationCategory(t))...)
	}
	if p.AutoSubscribe != nil && p.AutoSubscribe.Value != nil {
		for _, v := range *p.AutoSubscribe.Value {
			if !v.IsValid() {
				errs = append(errs, apierror.FieldError{Param: join(prefix, "auto_subscribe"), Code: apierror.CodeInvalidFormat, Message: fmt.Sprintf("%q isn't an auto-subscribe value.", v)})
			}
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return apierror.NewValidationError(errs[0].Message, errs...)
}

func validateSetting(param string, s Setting, t constants.NotificationCategory) []apierror.FieldError {
	var errs []apierror.FieldError
	for _, c := range s.Channels {
		if !c.IsValid() {
			errs = append(errs, apierror.FieldError{Param: param + ".channels", Code: apierror.CodeInvalidFormat, Message: fmt.Sprintf("%q isn't a channel.", c)})
		}
	}
	if s.Digest != "" && !s.Digest.IsValid() {
		errs = append(errs, apierror.FieldError{Param: param + ".digest", Code: apierror.CodeInvalidFormat, Message: fmt.Sprintf("%q isn't a digest.", s.Digest)})
	}
	if t.Required() && !s.Has(constants.NotificationChannelEmail) {
		errs = append(errs, apierror.FieldError{Param: param + ".channels", Code: apierror.CodeInvalidFormat, Message: fmt.Sprintf("%s notifications are always emailed; channels must include email.", t)})
	}
	return errs
}

func join(prefix, field string) string {
	if prefix == "" {
		return field
	}
	return prefix + "." + field
}
