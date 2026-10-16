package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/notifyprefs"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
)

// personalSettingOf reads a stored preference row as channels. Email counts only with a digest other than off.
func personalSettingOf(p *domain.NotificationPreference) notifyprefs.Setting {
	s := notifyprefs.Setting{Channels: []constants.NotificationChannel{}, Digest: constants.NotificationDigest(p.Digest)}
	if p.InAppEnabled {
		s.Channels = append(s.Channels, constants.NotificationChannelInApp)
	}
	if p.EmailEnabled && p.Digest != string(constants.NotificationDigestOff) {
		s.Channels = append(s.Channels, constants.NotificationChannelEmail)
	}
	if p.PushEnabled {
		s.Channels = append(s.Channels, constants.NotificationChannelPush)
	}
	if s.Digest == constants.NotificationDigestOff || s.Digest == "" {
		s.Digest = constants.NotificationDigestInstant
	}
	return s
}

func preferenceInputOf(category string, s notifyprefs.Setting) *domain.UpsertNotificationPreferenceInput {
	digest := s.Digest
	if digest == "" {
		digest = constants.NotificationDigestInstant
	}
	return &domain.UpsertNotificationPreferenceInput{
		Category:     category,
		InAppEnabled: s.Has(constants.NotificationChannelInApp),
		EmailEnabled: s.Has(constants.NotificationChannelEmail),
		PushEnabled:  s.Has(constants.NotificationChannelPush),
		Digest:       string(digest),
	}
}

// recipientSettings is everything that decides where one person's notifications go.
type recipientSettings struct {
	layers      notifyprefs.Layers
	own         *domain.NotificationSetting
	pausedUntil *time.Time
}

// paused reports whether email and push are held for the person now.
func (r recipientSettings) paused(now time.Time) bool {
	return r.pausedUntil != nil && r.pausedUntil.After(now)
}

// loadRecipientSettings reads a person's own settings, their role's defaults and the account's defaults. An empty
// accountUserID (an API key) has only the account's.
func loadRecipientSettings(ctx context.Context, f domain.RepoFactory, accountID, accountUserID string) (recipientSettings, *apierror.APIError) {
	settingRepo := f.NewNotificationSettingRepo()
	account, apiErr := settingRepo.Get(ctx, accountID, "")
	if apiErr != nil {
		return recipientSettings{}, apiErr
	}
	out := recipientSettings{layers: notifyprefs.Layers{Account: account.Defaults, Role: notifyprefs.Empty(), Personal: notifyprefs.Personal{Types: map[string]notifyprefs.Setting{}}}}
	if accountUserID == "" {
		return out, nil
	}
	rows, apiErr := f.NewNotificationPreferenceRepo().List(ctx, accountUserID)
	if apiErr != nil {
		return recipientSettings{}, apiErr
	}
	for _, row := range rows {
		s := personalSettingOf(row)
		if row.Category == "" {
			out.layers.Personal.Default = &s
		} else {
			out.layers.Personal.Types[row.Category] = s
		}
	}
	own, apiErr := settingRepo.Get(ctx, accountID, accountUserID)
	if apiErr != nil {
		return recipientSettings{}, apiErr
	}
	out.own = own
	out.layers.Personal.AutoSubscribe = own.AutoSubscribe
	out.pausedUntil = own.PausedUntil
	role, apiErr := settingRepo.RoleDefaults(ctx, accountUserID)
	if apiErr != nil {
		return recipientSettings{}, apiErr
	}
	out.layers.Role = role
	return out, nil
}

// settingsCache loads each recipient's settings once per fan-out.
type settingsCache struct {
	f         domain.RepoFactory
	accountID string
	byUser    map[string]recipientSettings
}

func newSettingsCache(f domain.RepoFactory, accountID string) *settingsCache {
	return &settingsCache{f: f, accountID: accountID, byUser: map[string]recipientSettings{}}
}

// get falls back to the built-in settings when a recipient's can't be read, so one bad row never blocks delivery.
func (c *settingsCache) get(ctx context.Context, accountUserID string) recipientSettings {
	if s, ok := c.byUser[accountUserID]; ok {
		return s
	}
	s, apiErr := loadRecipientSettings(ctx, c.f, c.accountID, accountUserID)
	if apiErr != nil {
		s = recipientSettings{layers: notifyprefs.Layers{Role: notifyprefs.Empty(), Account: notifyprefs.Empty()}}
	}
	c.byUser[accountUserID] = s
	return s
}

// effective is the channel decision for one notification type, with email and push held while the person is paused.
func (r recipientSettings) effective(t constants.NotificationCategory, now time.Time) domain.EffectiveNotificationPreference {
	s := r.layers.Resolve(t)
	eff := domain.EffectiveNotificationPreference{
		InAppEnabled: s.Has(constants.NotificationChannelInApp),
		EmailEnabled: s.Has(constants.NotificationChannelEmail),
		PushEnabled:  s.Has(constants.NotificationChannelPush),
		Digest:       string(s.Digest),
	}
	if r.paused(now) && !t.Required() {
		eff.EmailEnabled, eff.PushEnabled = false, false
	}
	return eff
}

func preferencesView(s recipientSettings) *domain.NotificationPreferences {
	types := make(map[string]notifyprefs.Setting, len(s.layers.Personal.Types))
	for k, v := range s.layers.Personal.Types {
		types[k] = v
	}
	return &domain.NotificationPreferences{
		Default:       s.layers.EffectiveDefault(),
		Types:         types,
		AutoSubscribe: s.layers.AutoSubscribe(),
		PausedUntil:   s.pausedUntil,
	}
}

func (s *messagingSvcImpl) GetNotificationPreferences(ctx context.Context) (*domain.NotificationPreferences, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.get_notification_preferences")
	defer span.End()
	identity, apiErr := s.actor(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	recipient, apiErr := s.recipient(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	settings, apiErr := loadRecipientSettings(ctx, s.repoFactory, identity.Target.AccountID, recipient)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return preferencesView(settings), nil
}

// preferencesAudit is a person's settings as their audit trail records them.
type preferencesAudit struct {
	Default       string `audit:"default"`
	Types         string `audit:"types"`
	AutoSubscribe string `audit:"auto_subscribe"`
	PausedUntil   string `audit:"paused_until"`
}

func auditOfPreferences(s recipientSettings) *preferencesAudit {
	a := &preferencesAudit{}
	if s.layers.Personal.Default != nil {
		a.Default = jsonString(s.layers.Personal.Default)
	}
	if len(s.layers.Personal.Types) > 0 {
		a.Types = jsonString(s.layers.Personal.Types)
	}
	if s.layers.Personal.AutoSubscribe != nil {
		a.AutoSubscribe = jsonString(s.layers.Personal.AutoSubscribe)
	}
	if s.pausedUntil != nil {
		a.PausedUntil = s.pausedUntil.UTC().Format(time.RFC3339)
	}
	return a
}

func jsonString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (s *messagingSvcImpl) UpdateNotificationPreferences(ctx context.Context, input domain.UpdateNotificationPreferencesInput) (*domain.NotificationPreferences, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.update_notification_preferences")
	defer span.End()

	identity, apiErr := s.actor(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}
	accountID := identity.Target.AccountID
	recipient, apiErr := s.recipient(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if recipient == "" {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("Notification preferences are only available to users with an account membership."))
	}
	if apiErr := input.Patch.Validate(""); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var view *domain.NotificationPreferences
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		before, apiErr := loadRecipientSettings(txCtx, f, accountID, recipient)
		if apiErr != nil {
			return apiErr
		}
		current := notifyprefs.Defaults{Default: before.layers.Personal.Default, Types: before.layers.Personal.Types, AutoSubscribe: before.layers.Personal.AutoSubscribe}
		next := current.Apply(input.Patch)

		prefRepo := f.NewNotificationPreferenceRepo()
		write := func(category string, setting *notifyprefs.Setting) *apierror.APIError {
			if setting == nil {
				return prefRepo.Delete(txCtx, recipient, category)
			}
			rowID, apiErr := id.GenID(id.NotificationPreferenceIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			return prefRepo.Upsert(txCtx, rowID, accountID, recipient, preferenceInputOf(category, *setting))
		}
		if input.Patch.Default != nil {
			if apiErr := write("", next.Default); apiErr != nil {
				return apiErr
			}
		}
		for t := range input.Patch.Types {
			var setting *notifyprefs.Setting
			if v, ok := next.Types[t]; ok {
				setting = &v
			}
			if apiErr := write(t, setting); apiErr != nil {
				return apiErr
			}
		}
		if input.Patch.AutoSubscribe != nil || input.PausedUntilSet {
			own := *before.own
			own.AutoSubscribe = next.AutoSubscribe
			if input.PausedUntilSet {
				own.PausedUntil = input.PausedUntil
			}
			if apiErr := f.NewNotificationSettingRepo().Upsert(txCtx, &own); apiErr != nil {
				return apiErr
			}
		}

		after, apiErr := loadRecipientSettings(txCtx, f, accountID, recipient)
		if apiErr != nil {
			return apiErr
		}
		view = preferencesView(after)
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeNotificationPreferences,
			ResourceID:   recipient,
			Changes:      audit.ComputeChanges(auditOfPreferences(before), auditOfPreferences(after)),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return view, nil
}

func (s *messagingSvcImpl) GetNotificationDefaults(ctx context.Context) (notifyprefs.Defaults, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.get_notification_defaults")
	defer span.End()
	identity, apiErr := s.actor(ctx)
	if apiErr != nil {
		return notifyprefs.Defaults{}, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionRead); apiErr != nil {
		return notifyprefs.Defaults{}, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return notifyprefs.Defaults{}, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}
	setting, apiErr := s.repoFactory.NewNotificationSettingRepo().Get(ctx, identity.Target.AccountID, "")
	if apiErr != nil {
		return notifyprefs.Defaults{}, tracing.Trace(span, apiErr)
	}
	return setting.Defaults, nil
}

// defaultsAudit is the account's notification defaults as the audit trail records them.
type defaultsAudit struct {
	Default       string `audit:"default"`
	Types         string `audit:"types"`
	AutoSubscribe string `audit:"auto_subscribe"`
}

func auditOfDefaults(d notifyprefs.Defaults) *defaultsAudit {
	a := &defaultsAudit{}
	if d.Default != nil {
		a.Default = jsonString(d.Default)
	}
	if len(d.Types) > 0 {
		a.Types = jsonString(d.Types)
	}
	if d.AutoSubscribe != nil {
		a.AutoSubscribe = jsonString(d.AutoSubscribe)
	}
	return a
}

func (s *messagingSvcImpl) UpdateNotificationDefaults(ctx context.Context, patch notifyprefs.Patch) (notifyprefs.Defaults, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.update_notification_defaults")
	defer span.End()
	identity, apiErr := s.actor(ctx)
	if apiErr != nil {
		return notifyprefs.Defaults{}, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAccount, types.ActionUpdate); apiErr != nil {
		return notifyprefs.Defaults{}, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return notifyprefs.Defaults{}, tracing.Trace(span, apierror.NewAuthenticationError("The OpenMRP-Account-ID header is required."))
	}
	if apiErr := patch.Validate(""); apiErr != nil {
		return notifyprefs.Defaults{}, tracing.Trace(span, apiErr)
	}
	accountID := identity.Target.AccountID
	var out notifyprefs.Defaults
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		repo := f.NewNotificationSettingRepo()
		setting, apiErr := repo.Get(txCtx, accountID, "")
		if apiErr != nil {
			return apiErr
		}
		before := setting.Defaults
		setting.Defaults = before.Apply(patch)
		if apiErr := repo.Upsert(txCtx, setting); apiErr != nil {
			return apiErr
		}
		out = setting.Defaults
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeNotificationDefaults,
			ResourceID:   accountID,
			Changes:      audit.ComputeChanges(auditOfDefaults(before), auditOfDefaults(out)),
		})
	})
	if apiErr != nil {
		return notifyprefs.Defaults{}, tracing.Trace(span, apiErr)
	}
	return out, nil
}
