package service

import (
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/notification-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestListConversationsByResource_PersonalTopicsOnlyListTheCallersConversations(t *testing.T) {
	t.Parallel()

	for _, resourceType := range []constants.ObjectType{constants.ObjectTypeConversation, constants.ObjectTypeAgentDefinition} {
		t.Run(string(resourceType), func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
			notifRepo.EXPECT().ResolveAccountUserID(gomock.Any(), "us_caller", "ac_team").Return("acus_caller", nil)

			mine := []*domain.Conversation{{ID: "cv_topic"}}
			convRepo := repositorymock.NewMockConversationRepo(ctrl)
			convRepo.EXPECT().
				ListParticipantByTopic(gomock.Any(), "ac_team", "acus_caller", string(resourceType), "anchor_1", int32(defaultNotificationPageSize)).
				Return(mine, nil)

			factory := factorymock.NewMockRepoFactory(ctrl)
			factory.EXPECT().NewNotificationRepo().Return(notifRepo)
			factory.EXPECT().NewConversationRepo().Return(convRepo)

			svc := &conversationSvcImpl{repoFactory: factory}
			got, apiErr := svc.ListConversationsByResource(groupCallerCtx(types.IdentityRelationTypeInternal, "messaging:read"), string(resourceType), "anchor_1", 0)

			require.Nil(t, apiErr)
			assert.Equal(t, mine, got)
		})
	}
}

func TestListConversationsByResource_RecordsListEveryAnchoredConversation(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	all := []*domain.Conversation{{ID: "cv_a"}, {ID: "cv_b"}}
	convRepo := repositorymock.NewMockConversationRepo(ctrl)
	convRepo.EXPECT().
		ListByResource(gomock.Any(), "ac_team", string(constants.ObjectTypeSalesOrder), "so_1", int32(defaultNotificationPageSize)).
		Return(all, nil)

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewConversationRepo().Return(convRepo)

	svc := &conversationSvcImpl{repoFactory: factory}
	got, apiErr := svc.ListConversationsByResource(groupCallerCtx(types.IdentityRelationTypeInternal, "messaging:read"), string(constants.ObjectTypeSalesOrder), "so_1", 0)

	require.Nil(t, apiErr)
	assert.Equal(t, all, got)
}
