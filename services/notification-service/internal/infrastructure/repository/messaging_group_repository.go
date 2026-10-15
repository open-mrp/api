package repository

import (
	"context"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/services/notification-service/internal/infrastructure/sqlc"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/tracing"
)

var messagingGroupRepoTracer = tracing.GetTracer("notification-service.messaging_group_repository")

type messagingGroupRepoImpl struct {
	db *sqlc.Queries
}

func NewMessagingGroupRepo(db *sqlc.Queries) domain.MessagingGroupRepo {
	return &messagingGroupRepoImpl{db: db}
}

func (r *messagingGroupRepoImpl) Create(ctx context.Context, g *domain.MessagingGroup) *apierror.APIError {
	ctx, span := messagingGroupRepoTracer.Start(ctx, "repository.messaging_group.create")
	defer span.End()
	err := r.db.CreateMessagingGroup(ctx, sqlc.CreateMessagingGroupParams{
		ID:                     g.ID,
		AccountID:              g.AccountID,
		Name:                   g.Name,
		CreatedByAccountUserID: db.NullStringPtr(g.CreatedByAccountUserID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *messagingGroupRepoImpl) Get(ctx context.Context, id, accountID string) (*domain.MessagingGroup, *apierror.APIError) {
	ctx, span := messagingGroupRepoTracer.Start(ctx, "repository.messaging_group.get")
	defer span.End()
	row, err := r.db.GetMessagingGroupByID(ctx, sqlc.GetMessagingGroupByIDParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.CodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return messagingGroupToDomain(row), nil
}

func (r *messagingGroupRepoImpl) ListPage(ctx context.Context, accountID string, page domain.KeysetPage) ([]*domain.MessagingGroup, *apierror.APIError) {
	ctx, span := messagingGroupRepoTracer.Start(ctx, "repository.messaging_group.list_page")
	defer span.End()
	var rows []sqlc.MessagingGroup
	var err error
	if page.CursorAt != nil {
		rows, err = r.db.ListMessagingGroupsPageAfter(ctx, sqlc.ListMessagingGroupsPageAfterParams{
			AccountID:       accountID,
			CursorUpdatedAt: *page.CursorAt,
			CursorID:        db.NullStringPtr(page.CursorID),
			Limit:           page.Limit,
		})
	} else {
		rows, err = r.db.ListMessagingGroupsPage(ctx, sqlc.ListMessagingGroupsPageParams{AccountID: accountID, Limit: page.Limit})
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.MessagingGroup, 0, len(rows))
	for _, row := range rows {
		out = append(out, messagingGroupToDomain(row))
	}
	return out, nil
}

func (r *messagingGroupRepoImpl) Rename(ctx context.Context, id, accountID, name string) (bool, *apierror.APIError) {
	ctx, span := messagingGroupRepoTracer.Start(ctx, "repository.messaging_group.rename")
	defer span.End()
	rows, err := r.db.UpdateMessagingGroupName(ctx, sqlc.UpdateMessagingGroupNameParams{Name: name, ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return rows > 0, nil
}

func (r *messagingGroupRepoImpl) Touch(ctx context.Context, id, accountID string) *apierror.APIError {
	ctx, span := messagingGroupRepoTracer.Start(ctx, "repository.messaging_group.touch")
	defer span.End()
	err := r.db.TouchMessagingGroup(ctx, sqlc.TouchMessagingGroupParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *messagingGroupRepoImpl) Delete(ctx context.Context, id, accountID string) (bool, *apierror.APIError) {
	ctx, span := messagingGroupRepoTracer.Start(ctx, "repository.messaging_group.delete")
	defer span.End()
	rows, err := r.db.DeleteMessagingGroup(ctx, sqlc.DeleteMessagingGroupParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return rows > 0, nil
}

func (r *messagingGroupRepoImpl) AddMember(ctx context.Context, m *domain.MessagingGroupMember) *apierror.APIError {
	ctx, span := messagingGroupRepoTracer.Start(ctx, "repository.messaging_group.add_member")
	defer span.End()
	err := r.db.CreateMessagingGroupMember(ctx, sqlc.CreateMessagingGroupMemberParams{
		ID:            m.ID,
		GroupID:       m.GroupID,
		AccountID:     m.AccountID,
		MemberType:    m.MemberType,
		AccountUserID: db.NullStringPtr(m.AccountUserID),
		AgentConfigID: db.NullStringPtr(m.AgentConfigID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *messagingGroupRepoImpl) ListMembers(ctx context.Context, groupID string) ([]*domain.MessagingGroupMember, *apierror.APIError) {
	ctx, span := messagingGroupRepoTracer.Start(ctx, "repository.messaging_group.list_members")
	defer span.End()
	rows, err := r.db.ListMessagingGroupMembers(ctx, groupID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.MessagingGroupMember, 0, len(rows))
	for _, row := range rows {
		out = append(out, messagingGroupMemberToDomain(row))
	}
	return out, nil
}

func (r *messagingGroupRepoImpl) RemoveMember(ctx context.Context, memberID, groupID string) (bool, *apierror.APIError) {
	ctx, span := messagingGroupRepoTracer.Start(ctx, "repository.messaging_group.remove_member")
	defer span.End()
	rows, err := r.db.DeleteMessagingGroupMemberByID(ctx, sqlc.DeleteMessagingGroupMemberByIDParams{ID: memberID, GroupID: groupID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return rows > 0, nil
}

func (r *messagingGroupRepoImpl) DeleteMembers(ctx context.Context, groupID string) *apierror.APIError {
	ctx, span := messagingGroupRepoTracer.Start(ctx, "repository.messaging_group.delete_members")
	defer span.End()
	err := r.db.DeleteMessagingGroupMembers(ctx, groupID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *messagingGroupRepoImpl) ClearConversationGroup(ctx context.Context, accountID, groupID string) *apierror.APIError {
	ctx, span := messagingGroupRepoTracer.Start(ctx, "repository.messaging_group.clear_conversation_group")
	defer span.End()
	err := r.db.ClearConversationGroup(ctx, sqlc.ClearConversationGroupParams{AccountID: accountID, GroupID: db.NullString(groupID)})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func messagingGroupToDomain(row sqlc.MessagingGroup) *domain.MessagingGroup {
	return &domain.MessagingGroup{
		ID:                     row.ID,
		AccountID:              row.AccountID,
		Name:                   row.Name,
		CreatedByAccountUserID: db.StringFromNullString(row.CreatedByAccountUserID),
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
	}
}

func messagingGroupMemberToDomain(row sqlc.MessagingGroupMember) *domain.MessagingGroupMember {
	return &domain.MessagingGroupMember{
		ID:            row.ID,
		GroupID:       row.GroupID,
		AccountID:     row.AccountID,
		MemberType:    row.MemberType,
		AccountUserID: db.StringFromNullString(row.AccountUserID),
		AgentConfigID: db.StringFromNullString(row.AgentConfigID),
		CreatedAt:     row.CreatedAt,
	}
}
