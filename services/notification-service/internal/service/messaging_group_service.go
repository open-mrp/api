package service

import (
	"context"
	"strings"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/idempotency"

	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

// CreateMessagingGroup creates a named roster of users and/or agents. User members are validated to
// exist; agent members are accepted by id (the same trust boundary AddAgentParticipant uses, since
// agent configs live in a separate database). The roster is created with its members in one tx.
func (s *conversationSvcImpl) CreateMessagingGroup(ctx context.Context, input domain.CreateMessagingGroupInput) (*domain.MessagingGroup, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.messaging_group.create")
	defer span.End()

	_, callerAcus, accountID, apiErr := s.caller(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, tracing.Trace(span, apierror.NewParameterMissingError("A group name is required.", "name"))
	}

	userIDs, agentIDs, apiErr := s.dedupeAndValidateMembers(ctx, input.MemberAccountUserIDs, input.MemberAgentConfigIDs)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Recovery-point idempotency: a roster has no natural dedup key, so a client retry would otherwise create a duplicate group. Scope on the client's Idempotency-Key.
	identity, _ := appctx.GetIdentityFromContext(ctx)
	idemKey, apiErr := upsertIdempotencyKey(ctx, s.repoFactory, identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	switch domain.RecoveryPoint(idemKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.MessagingGroup](ctx, idemKey.ResponseCode, idemKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error
	case domain.RecoveryPointStarted:
		groupID, apiErr := id.GenID(id.MessagingGroupIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		creator := callerAcus
		group := &domain.MessagingGroup{
			ID:                     groupID,
			AccountID:              accountID,
			Name:                   name,
			CreatedByAccountUserID: &creator,
		}
		var result *domain.MessagingGroup
		apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
			groupRepo := f.NewMessagingGroupRepo()
			if cErr := groupRepo.Create(txCtx, group); cErr != nil {
				return cErr
			}
			if sErr := seedGroupMembers(txCtx, groupRepo, groupID, accountID, userIDs, agentIDs); sErr != nil {
				return sErr
			}
			if apiErr := audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
				ServiceName:  domain.ServiceName,
				Action:       constants.AuditActionCreate,
				ResourceType: constants.ObjectTypeMessagingGroup,
				ResourceID:   group.ID,
				Changes:      audit.ComputeChanges(nil, group),
			}); apiErr != nil {
				return apiErr
			}
			// Re-read the persisted row so the response carries the DB-assigned created_at/updated_at (the in-memory group has zero-value timestamps), mirroring loadMessagingGroup which Update/AddMember/RemoveMember return through.
			loaded, gErr := groupRepo.Get(txCtx, groupID, accountID)
			if gErr != nil {
				return gErr
			}
			members, mErr := groupRepo.ListMembers(txCtx, groupID)
			if mErr != nil {
				return mErr
			}
			loaded.Members = members
			result = loaded
			return cacheSuccessResponse(txCtx, f, idemKey.TypeID, result)
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, cacheErrorResponse(ctx, s.repoFactory, idemKey.TypeID, apiErr))
		}
		return result, nil
	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idemKey.RecoveryPoint))
	}
}

func (s *conversationSvcImpl) ListMessagingGroups(ctx context.Context) ([]*domain.MessagingGroup, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.messaging_group.list")
	defer span.End()

	_, _, accountID, apiErr := s.caller(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	groupRepo := s.repoFactory.NewMessagingGroupRepo()
	groups, apiErr := groupRepo.List(ctx, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, g := range groups {
		members, apiErr := groupRepo.ListMembers(ctx, g.ID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		g.Members = members
	}
	return groups, nil
}

func (s *conversationSvcImpl) GetMessagingGroup(ctx context.Context, groupID string) (*domain.MessagingGroup, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.messaging_group.get")
	defer span.End()

	_, _, accountID, apiErr := s.caller(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.loadMessagingGroup(ctx, groupID, accountID)
}

func (s *conversationSvcImpl) UpdateMessagingGroup(ctx context.Context, groupID, name string) (*domain.MessagingGroup, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.messaging_group.update")
	defer span.End()

	_, _, accountID, apiErr := s.caller(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, tracing.Trace(span, apierror.NewParameterMissingError("A group name is required.", "name"))
	}
	// Capture the pre-image before the rename so the audit diff can surface the name change.
	old, apiErr := s.repoFactory.NewMessagingGroupRepo().Get(ctx, groupID, accountID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("The group was not found."))
		}
		return nil, tracing.Trace(span, apiErr)
	}
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		renamed, rErr := f.NewMessagingGroupRepo().Rename(txCtx, groupID, accountID, trimmed)
		if rErr != nil {
			return rErr
		}
		if !renamed {
			return apierror.NewResourceNotFoundError("The group was not found.")
		}
		updated := *old
		updated.Name = trimmed
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeMessagingGroup,
			ResourceID:   updated.ID,
			Changes:      audit.ComputeChanges(old, &updated),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.loadMessagingGroup(ctx, groupID, accountID)
}

func (s *conversationSvcImpl) DeleteMessagingGroup(ctx context.Context, groupID string) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.messaging_group.delete")
	defer span.End()

	_, _, accountID, apiErr := s.caller(ctx)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	// Load the pre-image before the delete for the audit ResourceID and change diff.
	existing, apiErr := s.repoFactory.NewMessagingGroupRepo().Get(ctx, groupID, accountID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			// Distinguish an already-deleted roster (410) from one that never existed (404).
			if wasDeleted, delErr := s.repoFactory.NewDeletedRecordRepo().Exists(ctx, constants.DeletedRecordResourceTypeMessagingGroup, groupID); delErr != nil {
				return tracing.Trace(span, delErr)
			} else if wasDeleted {
				return tracing.Trace(span, apierror.NewAlreadyDeletedError("This group has already been deleted and can no longer be modified."))
			}
			return tracing.Trace(span, apierror.NewResourceNotFoundError("The group was not found."))
		}
		return tracing.Trace(span, apiErr)
	}
	// Snapshot members onto the pre-image so the deleted_record captures the full roster for recovery.
	if members, mErr := s.repoFactory.NewMessagingGroupRepo().ListMembers(ctx, groupID); mErr == nil {
		existing.Members = members
	}
	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		groupRepo := f.NewMessagingGroupRepo()
		// Snapshot the roster (with members) into deleted_record before the hard delete.
		if drErr := f.NewDeletedRecordRepo().Create(txCtx, constants.DeletedRecordResourceTypeMessagingGroup, existing.ID, existing); drErr != nil {
			return drErr
		}
		deleted, dErr := groupRepo.Delete(txCtx, groupID, accountID)
		if dErr != nil {
			return dErr
		}
		if !deleted {
			return apierror.NewResourceNotFoundError("The group was not found.")
		}
		// Snapshot membership means conversations are functionally unaffected; only the provenance link is detached.
		if cErr := groupRepo.ClearConversationGroup(txCtx, accountID, groupID); cErr != nil {
			return cErr
		}
		// Remove member rows (the group row is gone; this keeps the member table clean — no FK cascade under relationMode=prisma).
		if mErr := groupRepo.DeleteMembers(txCtx, groupID); mErr != nil {
			return mErr
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeMessagingGroup,
			ResourceID:   existing.ID,
			Changes:      audit.ComputeChanges(existing, (*domain.MessagingGroup)(nil)),
		})
	})
}

func (s *conversationSvcImpl) AddMessagingGroupMember(ctx context.Context, input domain.AddMessagingGroupMemberInput) (*domain.MessagingGroup, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.messaging_group.add_member")
	defer span.End()

	_, _, accountID, apiErr := s.caller(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	groupRepo := s.repoFactory.NewMessagingGroupRepo()
	// The roster must exist in the caller's account.
	if _, apiErr := groupRepo.Get(ctx, input.GroupID, accountID); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("The group was not found."))
		}
		return nil, tracing.Trace(span, apiErr)
	}

	member := &domain.MessagingGroupMember{
		AccountID:  accountID,
		GroupID:    input.GroupID,
		MemberType: input.MemberType,
	}
	switch input.MemberType {
	case domain.MessagingGroupMemberTypeUser:
		if input.AccountUserID == "" {
			return nil, tracing.Trace(span, apierror.NewParameterMissingError("A member account_user_id is required.", "account_user_id"))
		}
		if _, apiErr := s.repoFactory.NewNotificationRepo().ResolveUserID(ctx, input.AccountUserID); apiErr != nil {
			if apiErr.Code == apierror.ErrorCodeResourceNotFound {
				return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The member does not exist.", "account_user_id"))
			}
			return nil, tracing.Trace(span, apiErr)
		}
		auid := input.AccountUserID
		member.AccountUserID = &auid
	case domain.MessagingGroupMemberTypeAgent:
		if input.AgentConfigID == "" {
			return nil, tracing.Trace(span, apierror.NewParameterMissingError("A member agent_config_id is required.", "agent_config_id"))
		}
		acid := input.AgentConfigID
		member.AgentConfigID = &acid
	default:
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The member type is invalid.", "member_type"))
	}

	memberID, apiErr := id.GenID(id.MessagingGroupMemberIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	member.ID = memberID
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txGroupRepo := f.NewMessagingGroupRepo()
		added := true
		if aErr := txGroupRepo.AddMember(txCtx, member); aErr != nil {
			// A duplicate (group, user/agent) is a benign no-op — the member is already present. db.MapSQLError maps the MySQL 1062 duplicate-key to ErrorCodeResourceExists.
			if aErr.Code != apierror.ErrorCodeResourceExists {
				return aErr
			}
			added = false
		}
		if tErr := txGroupRepo.Touch(txCtx, input.GroupID, accountID); tErr != nil {
			return tErr
		}
		// Only emit an audit event when a member row was actually inserted; a duplicate is a no-op.
		if !added {
			return nil
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeMessagingGroupMember,
			ResourceID:   member.ID,
			Changes:      audit.ComputeChanges(nil, member),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.loadMessagingGroup(ctx, input.GroupID, accountID)
}

func (s *conversationSvcImpl) RemoveMessagingGroupMember(ctx context.Context, groupID, memberID string) (*domain.MessagingGroup, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.messaging_group.remove_member")
	defer span.End()

	_, _, accountID, apiErr := s.caller(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	groupRepo := s.repoFactory.NewMessagingGroupRepo()
	if _, apiErr := groupRepo.Get(ctx, groupID, accountID); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("The group was not found."))
		}
		return nil, tracing.Trace(span, apiErr)
	}
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txGroupRepo := f.NewMessagingGroupRepo()
		removed, rErr := txGroupRepo.RemoveMember(txCtx, memberID, groupID)
		if rErr != nil {
			return rErr
		}
		if !removed {
			return apierror.NewResourceNotFoundError("The member was not found.")
		}
		if tErr := txGroupRepo.Touch(txCtx, groupID, accountID); tErr != nil {
			return tErr
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeMessagingGroupMember,
			ResourceID:   memberID,
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.loadMessagingGroup(ctx, groupID, accountID)
}

// loadMessagingGroup fetches a roster and its members, scoped to the caller's account.
func (s *conversationSvcImpl) loadMessagingGroup(ctx context.Context, groupID, accountID string) (*domain.MessagingGroup, *apierror.APIError) {
	groupRepo := s.repoFactory.NewMessagingGroupRepo()
	group, apiErr := groupRepo.Get(ctx, groupID, accountID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apierror.NewResourceNotFoundError("The group was not found.")
		}
		return nil, apiErr
	}
	members, apiErr := groupRepo.ListMembers(ctx, group.ID)
	if apiErr != nil {
		return nil, apiErr
	}
	group.Members = members
	return group, nil
}

// dedupeAndValidateMembers de-duplicates user/agent member ids and validates that each user exists.
// Agent ids are accepted as-is (their configs live in a separate database, like AddAgentParticipant).
func (s *conversationSvcImpl) dedupeAndValidateMembers(ctx context.Context, userIDs, agentIDs []string) ([]string, []string, *apierror.APIError) {
	users := make([]string, 0, len(userIDs))
	seenUser := map[string]struct{}{}
	for _, u := range userIDs {
		if u == "" {
			continue
		}
		if _, dup := seenUser[u]; dup {
			continue
		}
		seenUser[u] = struct{}{}
		if _, apiErr := s.repoFactory.NewNotificationRepo().ResolveUserID(ctx, u); apiErr != nil {
			if apiErr.Code == apierror.ErrorCodeResourceNotFound {
				return nil, nil, apierror.NewParameterInvalidError("A member does not exist.", "member_account_user_ids")
			}
			return nil, nil, apiErr
		}
		users = append(users, u)
	}
	agents := make([]string, 0, len(agentIDs))
	seenAgent := map[string]struct{}{}
	for _, a := range agentIDs {
		if a == "" {
			continue
		}
		if _, dup := seenAgent[a]; dup {
			continue
		}
		seenAgent[a] = struct{}{}
		agents = append(agents, a)
	}
	return users, agents, nil
}

// seedGroupMembers inserts the initial user + agent member rows for a roster.
func seedGroupMembers(ctx context.Context, groupRepo domain.MessagingGroupRepo, groupID, accountID string, userIDs, agentIDs []string) *apierror.APIError {
	for _, u := range userIDs {
		mid, genErr := id.GenID(id.MessagingGroupMemberIDPrefix, nil)
		if genErr != nil {
			return genErr
		}
		uid := u
		if aErr := groupRepo.AddMember(ctx, &domain.MessagingGroupMember{
			ID:            mid,
			GroupID:       groupID,
			AccountID:     accountID,
			MemberType:    domain.MessagingGroupMemberTypeUser,
			AccountUserID: &uid,
		}); aErr != nil {
			return aErr
		}
	}
	for _, a := range agentIDs {
		mid, genErr := id.GenID(id.MessagingGroupMemberIDPrefix, nil)
		if genErr != nil {
			return genErr
		}
		acid := a
		if aErr := groupRepo.AddMember(ctx, &domain.MessagingGroupMember{
			ID:            mid,
			GroupID:       groupID,
			AccountID:     accountID,
			MemberType:    domain.MessagingGroupMemberTypeAgent,
			AgentConfigID: &acid,
		}); aErr != nil {
			return aErr
		}
	}
	return nil
}
