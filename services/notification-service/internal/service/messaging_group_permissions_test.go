package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/notification-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func groupCallerCtx(relation types.IdentityRelationType, perms ...string) context.Context {
	acct := "ac_team"
	roleType := string(constants.RoleTypeCustom)
	granted := map[string]bool{}
	for _, p := range perms {
		granted[p] = true
	}
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: "ac_team"},
		Actor: &types.IdentityActor{
			RelationType: relation,
			ID:           "us_caller",
			AccountID:    &acct,
			RoleType:     &roleType,
			Permissions:  granted,
		},
	})
}

// Every operation must be refused before it touches a repository when the caller lacks its messaging permission.
func TestMessagingGroups_RequireTheirMessagingPermission(t *testing.T) {
	t.Parallel()

	ops := []struct {
		name string
		perm string
		call func(*conversationSvcImpl, context.Context) *apierror.APIError
	}{
		{"create", "messaging:create", func(s *conversationSvcImpl, ctx context.Context) *apierror.APIError {
			_, e := s.CreateMessagingGroup(ctx, domain.CreateMessagingGroupInput{Name: "Floor"})
			return e
		}},
		{"list", "messaging:read", func(s *conversationSvcImpl, ctx context.Context) *apierror.APIError {
			_, e := s.ListMessagingGroups(ctx)
			return e
		}},
		{"get", "messaging:read", func(s *conversationSvcImpl, ctx context.Context) *apierror.APIError {
			_, e := s.GetMessagingGroup(ctx, "mgrp_x")
			return e
		}},
		{"update", "messaging:update", func(s *conversationSvcImpl, ctx context.Context) *apierror.APIError {
			_, e := s.UpdateMessagingGroup(ctx, "mgrp_x", "Renamed")
			return e
		}},
		{"delete", "messaging:delete", func(s *conversationSvcImpl, ctx context.Context) *apierror.APIError {
			return s.DeleteMessagingGroup(ctx, "mgrp_x")
		}},
		{"add member", "messaging:update", func(s *conversationSvcImpl, ctx context.Context) *apierror.APIError {
			_, e := s.AddMessagingGroupMember(ctx, domain.AddMessagingGroupMemberInput{GroupID: "mgrp_x", MemberType: domain.MessagingGroupMemberTypeUser, AccountUserID: "acus_x"})
			return e
		}},
		{"remove member", "messaging:update", func(s *conversationSvcImpl, ctx context.Context) *apierror.APIError {
			_, e := s.RemoveMessagingGroupMember(ctx, "mgrp_x", "mgm_x")
			return e
		}},
	}

	for _, op := range ops {
		t.Run(op.name+" without permission", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			svc := &conversationSvcImpl{repoFactory: factorymock.NewMockRepoFactory(ctrl)}

			// Holds every messaging permission except the one this operation needs.
			var others []string
			for _, p := range []string{"messaging:create", "messaging:read", "messaging:update", "messaging:delete"} {
				if p != op.perm {
					others = append(others, p)
				}
			}
			apiErr := op.call(svc, groupCallerCtx(types.IdentityRelationTypeInternal, others...))

			require.NotNil(t, apiErr)
			assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code)
		})

		t.Run(op.name+" by a customer actor", func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			svc := &conversationSvcImpl{repoFactory: factorymock.NewMockRepoFactory(ctrl)}

			apiErr := op.call(svc, groupCallerCtx(types.IdentityRelationTypeCustomer, op.perm))

			require.NotNil(t, apiErr)
			assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code)
		})
	}
}

func TestMessagingGroups_PermittedCallerProceeds(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	notifRepo.EXPECT().ResolveAccountUserID(gomock.Any(), "us_caller", "ac_team").Return("acus_caller", nil)
	groupRepo := repositorymock.NewMockMessagingGroupRepo(ctrl)
	groupRepo.EXPECT().List(gomock.Any(), "ac_team").Return(nil, nil)
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewNotificationRepo().Return(notifRepo)
	factory.EXPECT().NewMessagingGroupRepo().Return(groupRepo)
	svc := &conversationSvcImpl{repoFactory: factory}

	groups, apiErr := svc.ListMessagingGroups(groupCallerCtx(types.IdentityRelationTypeInternal, "messaging:read"))

	require.Nil(t, apiErr)
	assert.Empty(t, groups)
}
