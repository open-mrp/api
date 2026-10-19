package service

import (
	"context"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/notifyprefs"
	apierror "github.com/open-mrp/apikit/apierror"

	"go.uber.org/mock/gomock"
)

// stubNotificationSettings serves the given account and role defaults, and no personal settings row.
func stubNotificationSettings(ctrl *gomock.Controller, factory *factorymock.MockRepoFactory, account, role notifyprefs.Defaults) *repositorymock.MockNotificationSettingRepo {
	repo := repositorymock.NewMockNotificationSettingRepo(ctrl)
	repo.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, accountID, accountUserID string) (*domain.NotificationSetting, *apierror.APIError) {
			s := &domain.NotificationSetting{AccountID: accountID, AccountUserID: accountUserID, Defaults: notifyprefs.Empty()}
			if accountUserID == "" {
				s.Defaults = account
			}
			return s, nil
		}).AnyTimes()
	repo.EXPECT().RoleDefaults(gomock.Any(), gomock.Any()).Return(role, nil).AnyTimes()
	factory.EXPECT().NewNotificationSettingRepo().Return(repo).AnyTimes()
	return repo
}

// stubDefaultNotificationSettings leaves everyone on the built-in settings.
func stubDefaultNotificationSettings(ctrl *gomock.Controller, factory *factorymock.MockRepoFactory) {
	stubNotificationSettings(ctrl, factory, notifyprefs.Empty(), notifyprefs.Empty())
	prefRepo := repositorymock.NewMockNotificationPreferenceRepo(ctrl)
	prefRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	factory.EXPECT().NewNotificationPreferenceRepo().Return(prefRepo).AnyTimes()
}
