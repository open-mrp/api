package repository

import (
	"context"
	"database/sql"
	"slices"
	"strconv"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

var accountUserRepoTracer = tracing.GetTracer("core-service.account_user_repository")

type accountUserRepoImpl struct {
	queries *sqlc.Queries
}

func NewAccountUserRepo(queries *sqlc.Queries) domain.AccountUserRepo {
	return &accountUserRepoImpl{queries: queries}
}

func (r *accountUserRepoImpl) FindByAccountAndUserID(ctx context.Context, userID, accountID string) (*domain.AccountUser, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.find_by_account_and_user_id")
	defer span.End()

	row, err := r.queries.FindAccountUserWithRoleByAccountIDAndUserID(ctx, sqlc.FindAccountUserWithRoleByAccountIDAndUserIDParams{
		AccountID: accountID,
		UserID:    userID,
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.AccountUser{
		ID:           row.ID,
		UserID:       row.UserID,
		DepartmentID: db.StringFromNullString(row.DepartmentID),
		RoleID:       db.StringFromNullString(row.RoleID),
		RoleType:     db.StringFromNullString(row.RoleTypeCode),
		RoleName:     db.StringFromNullString(row.RoleName),
		AccountID:    row.AccountID,
		LastUsedAt:   db.TimeFromNullTime(row.LastUsedAt),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
}

func (r *accountUserRepoImpl) ResolveAccountUserID(ctx context.Context, accountID, userOrAccountUserID string) (string, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.resolve_account_user_id")
	defer span.End()

	id, err := r.queries.ResolveAccountUserID(ctx, sqlc.ResolveAccountUserIDParams{
		AccountID:           accountID,
		UserOrAccountUserID: userOrAccountUserID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return "", apiErr
		}
		return "", tracing.Trace(span, apiErr)
	}

	return id, nil
}

func (r *accountUserRepoImpl) FindAffiliationsByUserID(ctx context.Context, userID string) ([]domain.AccountAffiliation, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.find_affiliations_by_user_id")
	defer span.End()

	rows, err := r.queries.FindAccountAffiliationsByUserID(ctx, userID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	affiliations := make([]domain.AccountAffiliation, len(rows))
	for i, row := range rows {
		affiliations[i] = domain.AccountAffiliation{
			AccountID:   row.AccountID,
			AccountName: row.AccountName,
			RoleID:      row.RoleID,
			RoleName:    row.RoleName,
			RoleType:    row.RoleTypeCode,
			LastUsedAt:  db.TimeFromNullTime(row.LastUsedAt),
		}
	}

	return affiliations, nil
}

func (r *accountUserRepoImpl) FindLastUsedAccountID(ctx context.Context, userID string) (string, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.find_last_used_account_id")
	defer span.End()

	accountID, err := r.queries.FindLastUsedAccountID(ctx, userID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return "", nil
		}
		return "", tracing.Trace(span, apiErr)
	}

	return accountID, nil
}

func (r *accountUserRepoImpl) UpdateLastUsedAt(ctx context.Context, accountUserID string, lastUsedAt time.Time) *apierror.APIError {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.update_last_used_at")
	defer span.End()

	err := r.queries.UpdateAccountUserLastUsedAt(ctx, sqlc.UpdateAccountUserLastUsedAtParams{
		ID:         accountUserID,
		LastUsedAt: db.NullTime(lastUsedAt),
	})

	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountUserRepoImpl) GetAdminRoleID(ctx context.Context) (string, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.get_admin_role_id")
	defer span.End()

	roleID, err := r.queries.GetAdminRoleID(ctx)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return roleID, nil
}

func (r *accountUserRepoImpl) DeactivateExcept(ctx context.Context, accountID, keepUserID string, limit int32) (int64, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.deactivate_except")
	defer span.End()

	result, err := r.queries.DeactivateAccountUsersExcept(ctx, sqlc.DeactivateAccountUsersExceptParams{
		AccountID: accountID,
		UserID:    keepUserID,
		Limit:     limit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	rows, _ := result.RowsAffected()
	return rows, nil
}

func (r *accountUserRepoImpl) EnsureActive(ctx context.Context, accountID, userID string) *apierror.APIError {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.ensure_active")
	defer span.End()

	_, err := r.queries.EnsureAccountUserActive(ctx, sqlc.EnsureAccountUserActiveParams{
		AccountID: accountID,
		UserID:    userID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountUserRepoImpl) CountActive(ctx context.Context, accountID string) (int64, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.count_active")
	defer span.End()

	count, err := r.queries.CountActiveAccountUsers(ctx, accountID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return count, nil
}

func (r *accountUserRepoImpl) CountByRoleID(ctx context.Context, accountID, roleID string) (int64, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.count_by_role_id")
	defer span.End()

	count, err := r.queries.CountAccountUsersByRoleID(ctx, sqlc.CountAccountUsersByRoleIDParams{
		RoleID:    db.NullString(roleID),
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return count, nil
}

func (r *accountUserRepoImpl) ReactivateUsers(ctx context.Context, accountID string, limit int32) (int64, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.reactivate_users")
	defer span.End()

	result, err := r.queries.ReactivateAccountUsers(ctx, sqlc.ReactivateAccountUsersParams{
		AccountID: accountID,
		Limit:     limit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	rows, _ := result.RowsAffected()
	return rows, nil
}

func (r *accountUserRepoImpl) List(ctx context.Context, params domain.ListAccountUsersParams) (*domain.ListAccountUsersResult, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.list")
	defer span.End()

	searchQuery, queryLike := buildAccountUserSearchParams(params.Query)

	countResult, err := r.queries.CountAccountUsersFiltered(ctx, sqlc.CountAccountUsersFilteredParams{
		AccountID:            params.AccountID,
		IncludeRemoved:       params.IncludeRemoved,
		RoleType:             db.NullStringPtr(params.RoleType),
		Query:                searchQuery,
		QueryLike:            queryLike,
		IsCommissionEligible: nullBoolFromPtr(params.IsCommissionEligible),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	base := sqlc.ListAccountUsersForwardBaseParams{
		AccountID:            params.AccountID,
		IncludeRemoved:       params.IncludeRemoved,
		RoleType:             db.NullStringPtr(params.RoleType),
		Query:                searchQuery,
		QueryLike:            queryLike,
		Limit:                params.Limit + 1,
		IsCommissionEligible: nullBoolFromPtr(params.IsCommissionEligible),
	}

	var cursorDir *pagination.Direction
	var items []*domain.AccountUserDetail
	var pageInfo pagination.PageInfo

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			brows, err := r.queries.ListAccountUsersBackwardBase(ctx, sqlc.ListAccountUsersBackwardBaseParams{
				AccountID:            base.AccountID,
				IncludeRemoved:       base.IncludeRemoved,
				RoleType:             base.RoleType,
				IsCommissionEligible: base.IsCommissionEligible,
				Query:                base.Query,
				QueryLike:            base.QueryLike,
				CursorCreatedAt:      cur.OccurredAt,
				CursorID:             cur.ID,
				Limit:                base.Limit,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			details := make([]*domain.AccountUserDetail, len(brows))
			for i := range brows {
				details[i] = mapAccountUserBaseBackwardRow(brows[i])
			}
			items, pageInfo = pagination.BuildPageString(details, params.Limit, cursorDir, accountUserDetailCreatedAt, accountUserDetailID)
		} else {
			frows, err := r.queries.ListAccountUsersForwardBase(ctx, sqlc.ListAccountUsersForwardBaseParams{
				AccountID:            base.AccountID,
				IncludeRemoved:       base.IncludeRemoved,
				RoleType:             base.RoleType,
				IsCommissionEligible: base.IsCommissionEligible,
				Query:                base.Query,
				QueryLike:            base.QueryLike,
				CursorCreatedAt:      sql.NullTime{Time: cur.OccurredAt, Valid: true},
				CursorID:             sql.NullString{String: cur.ID, Valid: true},
				Limit:                base.Limit,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			details := make([]*domain.AccountUserDetail, len(frows))
			for i := range frows {
				details[i] = mapAccountUserBaseForwardRow(frows[i])
			}
			items, pageInfo = pagination.BuildPageString(details, params.Limit, cursorDir, accountUserDetailCreatedAt, accountUserDetailID)
		}
	} else {
		frows, err := r.queries.ListAccountUsersForwardBase(ctx, base)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		details := make([]*domain.AccountUserDetail, len(frows))
		for i := range frows {
			details[i] = mapAccountUserBaseForwardRow(frows[i])
		}
		items, pageInfo = pagination.BuildPageString(details, params.Limit, cursorDir, accountUserDetailCreatedAt, accountUserDetailID)
	}

	if apiErr := r.stitchAccountUsers(ctx, items, params.Includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.ListAccountUsersResult{
		Items:      items,
		PageInfo:   pageInfo,
		TotalCount: countResult,
	}, nil
}

func buildAccountUserSearchParams(query *string) (sql.NullString, any) {
	if query == nil || *query == "" {
		return sql.NullString{}, nil
	}
	ft := db.NewFulltextSearch(query)
	queryLike := accountUserQueryLike(query)
	if ft.Fulltext.Valid {
		return ft.Fulltext, queryLike
	}
	if ft.Like.Valid {
		// LIKE-only (short query): use a non-null guard so username/email LIKE runs.
		return sql.NullString{String: "", Valid: true}, queryLike
	}
	return sql.NullString{}, nil
}

func accountUserQueryLike(query *string) any {
	if query == nil {
		return nil
	}
	if *query == "" {
		return nil
	}
	return db.EscapeLike(*query)
}

func accountUserDetailCreatedAt(d *domain.AccountUserDetail) time.Time { return d.CreatedAt }
func accountUserDetailID(d *domain.AccountUserDetail) string           { return d.ID }

func mapAccountUserBaseForwardRow(row sqlc.ListAccountUsersForwardBaseRow) *domain.AccountUserDetail {
	return &domain.AccountUserDetail{
		ID:                   row.ID,
		UserID:               row.UserID,
		Name:                 db.StringFromNullString(row.Name),
		Email:                db.StringFromNullString(row.Email),
		Username:             db.StringFromNullString(row.Username),
		ImageURL:             db.StringFromNullString(row.ImageUrl),
		EmailVerified:        row.EmailVerified.Valid,
		RoleID:               db.StringFromNullString(row.RoleID),
		DepartmentID:         db.StringFromNullString(row.DepartmentID),
		StatusCode:           constants.AccountUserStatus(row.StatusCode),
		IsCommissionEligible: row.IsCommissionEligible,
		LastUsedAt:           db.TimeFromNullTime(row.LastUsedAt),
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func mapAccountUserBaseBackwardRow(row sqlc.ListAccountUsersBackwardBaseRow) *domain.AccountUserDetail {
	return &domain.AccountUserDetail{
		ID:                   row.ID,
		UserID:               row.UserID,
		Name:                 db.StringFromNullString(row.Name),
		Email:                db.StringFromNullString(row.Email),
		Username:             db.StringFromNullString(row.Username),
		ImageURL:             db.StringFromNullString(row.ImageUrl),
		EmailVerified:        row.EmailVerified.Valid,
		RoleID:               db.StringFromNullString(row.RoleID),
		DepartmentID:         db.StringFromNullString(row.DepartmentID),
		StatusCode:           constants.AccountUserStatus(row.StatusCode),
		IsCommissionEligible: row.IsCommissionEligible,
		LastUsedAt:           db.TimeFromNullTime(row.LastUsedAt),
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func mapAccountUserBaseDetailRow(row sqlc.GetAccountUserDetailBaseRow) *domain.AccountUserDetail {
	return &domain.AccountUserDetail{
		ID:                   row.ID,
		UserID:               row.UserID,
		Name:                 db.StringFromNullString(row.Name),
		Email:                db.StringFromNullString(row.Email),
		Username:             db.StringFromNullString(row.Username),
		ImageURL:             db.StringFromNullString(row.ImageUrl),
		EmailVerified:        row.EmailVerified.Valid,
		RoleID:               db.StringFromNullString(row.RoleID),
		DepartmentID:         db.StringFromNullString(row.DepartmentID),
		StatusCode:           constants.AccountUserStatus(row.StatusCode),
		IsCommissionEligible: row.IsCommissionEligible,
		LastUsedAt:           db.TimeFromNullTime(row.LastUsedAt),
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func mapAccountUserBaseByIDRow(row sqlc.GetAccountUserDetailBaseByIDRow) *domain.AccountUserDetail {
	detail := &domain.AccountUserDetail{
		ID:                   row.ID,
		UserID:               row.UserID,
		Name:                 db.StringFromNullString(row.Name),
		Email:                db.StringFromNullString(row.Email),
		Username:             db.StringFromNullString(row.Username),
		ImageURL:             db.StringFromNullString(row.ImageUrl),
		EmailVerified:        row.EmailVerified.Valid,
		RoleID:               db.StringFromNullString(row.RoleID),
		DepartmentID:         db.StringFromNullString(row.DepartmentID),
		DepartmentName:       db.StringFromNullString(row.DepartmentName),
		StatusCode:           constants.AccountUserStatus(row.StatusCode),
		IsCommissionEligible: row.IsCommissionEligible,
		LastUsedAt:           db.TimeFromNullTime(row.LastUsedAt),
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
	detail.RoleType = db.StringFromNullString(row.RoleTypeCode)
	if row.DepartmentCreatedAt.Valid {
		detail.DepartmentCreatedAt = &row.DepartmentCreatedAt.Time
	}
	if row.DepartmentUpdatedAt.Valid {
		detail.DepartmentUpdatedAt = &row.DepartmentUpdatedAt.Time
	}
	return detail
}

func (r *accountUserRepoImpl) GetDetail(ctx context.Context, accountID, userID string, includes []string) (*domain.AccountUserDetail, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.get_detail")
	defer span.End()

	row, err := r.queries.GetAccountUserDetailBase(ctx, sqlc.GetAccountUserDetailBaseParams{
		AccountID: accountID,
		UserID:    userID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	detail := mapAccountUserBaseDetailRow(row)
	if apiErr := r.stitchAccountUsers(ctx, []*domain.AccountUserDetail{detail}, includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return detail, nil
}

func (r *accountUserRepoImpl) GetDetailByAccountAndID(ctx context.Context, accountID, accountUserID string, includes []string) (*domain.AccountUserDetail, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.get_detail_by_account_and_id")
	defer span.End()

	row, err := r.queries.GetAccountUserDetailBaseByID(ctx, sqlc.GetAccountUserDetailBaseByIDParams{
		AccountID: accountID,
		ID:        accountUserID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	detail := mapAccountUserBaseByIDRow(row)
	if apiErr := r.stitchAccountUsers(ctx, []*domain.AccountUserDetail{detail}, includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return detail, nil
}

func (r *accountUserRepoImpl) Create(ctx context.Context, id, accountID, userID string, roleID, departmentID *string, isCommissionEligible bool) *apierror.APIError {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.create")
	defer span.End()

	err := r.queries.InsertAccountUser(ctx, sqlc.InsertAccountUserParams{
		ID:                   id,
		AccountID:            accountID,
		UserID:               userID,
		RoleID:               db.NullStringPtr(roleID),
		DepartmentID:         db.NullStringPtr(departmentID),
		IsCommissionEligible: isCommissionEligible,
		StatusCode:           "active",
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountUserRepoImpl) Update(ctx context.Context, accountUserID string, roleID, departmentID *string, isCommissionEligible bool) *apierror.APIError {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.update")
	defer span.End()

	err := r.queries.UpdateAccountUserRoleAndDepartment(ctx, sqlc.UpdateAccountUserRoleAndDepartmentParams{
		ID:                   accountUserID,
		RoleID:               db.NullStringPtr(roleID),
		DepartmentID:         db.NullStringPtr(departmentID),
		IsCommissionEligible: isCommissionEligible,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountUserRepoImpl) ReactivateRemovedAccountUser(ctx context.Context, accountID, userID string, roleID, departmentID *string, isCommissionEligible bool) (string, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.reactivate_removed")
	defer span.End()

	removedID, err := r.queries.FindRemovedAccountUserIDByAccountAndUserID(ctx, sqlc.FindRemovedAccountUserIDByAccountAndUserIDParams{
		AccountID: accountID,
		UserID:    userID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	if err := r.queries.ReactivateRemovedAccountUser(ctx, sqlc.ReactivateRemovedAccountUserParams{
		RoleID:               db.NullStringPtr(roleID),
		DepartmentID:         db.NullStringPtr(departmentID),
		IsCommissionEligible: isCommissionEligible,
		AccountID:            accountID,
		UserID:               userID,
	}); err != nil {
		return "", tracing.Trace(span, db.MapSQLError(err))
	}

	return removedID, nil
}

func (r *accountUserRepoImpl) SoftDelete(ctx context.Context, accountUserID string) *apierror.APIError {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.soft_delete")
	defer span.End()

	result, err := r.queries.SoftDeleteAccountUser(ctx, accountUserID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Account user not found."))
	}

	return nil
}

func (r *accountUserRepoImpl) UpdateStatus(ctx context.Context, accountUserID string, status constants.AccountUserStatus) *apierror.APIError {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.update_status")
	defer span.End()

	err := r.queries.UpdateAccountUserStatus(ctx, sqlc.UpdateAccountUserStatusParams{
		ID:         accountUserID,
		StatusCode: string(status),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountUserRepoImpl) RevokeRefreshTokensByUserID(ctx context.Context, userID string) *apierror.APIError {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.revoke_refresh_tokens")
	defer span.End()

	err := r.queries.RevokeRefreshTokensByUserID(ctx, userID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountUserRepoImpl) FindFirstAccountIDByUserID(ctx context.Context, userID string) (string, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.find_first_account_id_by_user_id")
	defer span.End()

	accountID, err := r.queries.FindFirstAccountIDByUserID(ctx, userID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return "", tracing.Trace(span, apiErr)
	}

	return accountID, nil
}

func (r *accountUserRepoImpl) stitchAccountUsers(ctx context.Context, items []*domain.AccountUserDetail, incs []string) *apierror.APIError {
	if len(items) == 0 {
		return nil
	}

	if slices.Contains(incs, "role") {
		roleIDs := make([]string, 0, len(items))
		for _, au := range items {
			if au.RoleID != nil {
				roleIDs = append(roleIDs, *au.RoleID)
			}
		}
		if len(roleIDs) > 0 {
			rows, err := r.queries.GetRolesByIDs(ctx, roleIDs)
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return apiErr
			}
			roleMap := make(map[string]sqlc.GetRolesByIDsRow, len(rows))
			for _, row := range rows {
				roleMap[row.ID] = row
			}
			for _, au := range items {
				if au.RoleID != nil {
					if role, ok := roleMap[*au.RoleID]; ok {
						au.RoleName = &role.Name
						typeCode := role.RoleTypeCode
						au.RoleType = &typeCode
					}
				}
			}
		}
	}

	if slices.Contains(incs, "department") {
		deptIDs := make([]string, 0, len(items))
		for _, au := range items {
			if au.DepartmentID != nil {
				deptIDs = append(deptIDs, *au.DepartmentID)
			}
		}
		if len(deptIDs) > 0 {
			rows, err := r.queries.GetDepartmentsByIDs(ctx, deptIDs)
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return apiErr
			}
			deptMap := make(map[string]sqlc.GetDepartmentsByIDsRow, len(rows))
			for _, row := range rows {
				deptMap[row.ID] = row
			}
			for _, au := range items {
				if au.DepartmentID != nil {
					if dept, ok := deptMap[*au.DepartmentID]; ok {
						au.DepartmentName = &dept.Name
						au.DepartmentCreatedAt = &dept.CreatedAt
						au.DepartmentUpdatedAt = &dept.UpdatedAt
					}
				}
			}
		}
	}

	return nil
}

func (r *accountUserRepoImpl) FindTenancyAccountsByUserID(ctx context.Context, userID string) ([]domain.TenancyAccount, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.find_tenancy_accounts_by_user_id")
	defer span.End()

	rows, err := r.queries.FindTenancyAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to find tenancy accounts by user ID."))
	}

	accounts := make([]domain.TenancyAccount, len(rows))
	for i, row := range rows {
		accounts[i] = domain.TenancyAccount{
			AccountID:                row.AccountID,
			AccountName:              row.AccountName,
			AccountTypeCode:          row.AccountTypeCode,
			OnboardingStatusCode:     row.OnboardingStatusCode,
			PlanCode:                 row.PlanCode,
			AccountUserID:            row.AccountUserID,
			AccountUserStatusCode:    row.AccountUserStatusCode,
			LastUsedAt:               db.TimeFromNullTime(row.LastUsedAt),
			RoleID:                   db.StringFromNullString(row.RoleID),
			RoleName:                 db.StringFromNullString(row.RoleName),
			RoleType:                 db.StringFromNullString(row.RoleTypeCode),
			RoleCreatedAt:            db.TimeFromNullTime(row.RoleCreatedAt),
			RoleUpdatedAt:            db.TimeFromNullTime(row.RoleUpdatedAt),
			OwnerAccountID:           db.StringFromNullString(row.OwnerAccountID),
			InternalStripeCustomerID: db.StringFromNullString(row.InternalStripeCustomerID),
			Plan:                     buildTenancyAccountPlanSummary(row),
		}
	}

	return accounts, nil
}

func buildTenancyAccountPlanSummary(row sqlc.FindTenancyAccountsByUserIDRow) *domain.TenancyAccountPlanSummary {
	if !row.PlanTypeID.Valid {
		return nil
	}

	plan := &domain.TenancyAccountPlanSummary{
		TypeID:       row.PlanTypeID.String,
		Name:         row.PlanName.String,
		PlanTypeCode: row.PlanPlanTypeCode.String,
		Version:      row.PlanVersion.Int32,
	}

	if row.PlanPricePerSeat.Valid {
		if v, err := strconv.ParseFloat(row.PlanPricePerSeat.String, 64); err == nil {
			plan.PricePerSeat = v
		}
	}

	if row.PlanPricePerMonth.Valid {
		if v, err := strconv.ParseFloat(row.PlanPricePerMonth.String, 64); err == nil {
			plan.PricePerMonth = &v
		}
	}

	if row.PlanSeatMinimum.Valid {
		min := row.PlanSeatMinimum.Int32
		plan.SeatMinimum = &min
	}

	return plan
}

func (r *accountUserRepoImpl) MarkUsedByAccountAndUser(ctx context.Context, accountID, userID string) *apierror.APIError {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.mark_used_by_account_and_user")
	defer span.End()

	err := r.queries.MarkUsedByAccountAndUser(ctx, sqlc.MarkUsedByAccountAndUserParams{
		AccountID: accountID,
		UserID:    userID,
	})
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to mark account user as used."))
	}

	return nil
}

func (r *accountUserRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.AccountUserDetail, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.get_by_ids")
	defer span.End()

	rows, err := r.queries.GetAccountUserDetailsByIDs(ctx, sqlc.GetAccountUserDetailsByIDsParams{
		Ids:       ids,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	items := make([]*domain.AccountUserDetail, len(rows))
	for i, row := range rows {
		au := &domain.AccountUserDetail{
			ID:         row.ID,
			UserID:     row.UserID,
			StatusCode: constants.AccountUserStatus(row.StatusCode),
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		}
		if row.Name.Valid {
			au.Name = &row.Name.String
		}
		if row.Email.Valid {
			au.Email = &row.Email.String
		}
		if row.Username.Valid {
			au.Username = &row.Username.String
		}
		if row.ImageUrl.Valid {
			au.ImageURL = &row.ImageUrl.String
		}
		if row.EmailVerified.Valid {
			au.EmailVerified = true
		}
		if row.RoleID.Valid {
			au.RoleID = &row.RoleID.String
		}
		if row.DepartmentID.Valid {
			au.DepartmentID = &row.DepartmentID.String
		}
		if row.LastUsedAt.Valid {
			au.LastUsedAt = &row.LastUsedAt.Time
		}
		au.IsCommissionEligible = row.IsCommissionEligible
		items[i] = au
	}

	return items, nil
}

func nullBoolFromPtr(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *b, Valid: true}
}
