package repository

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
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
		RoleTypeCode: db.StringFromNullString(row.RoleTypeCode),
		AccountID:    row.AccountID,
		LastUsedAt:   db.TimeFromNullTime(row.LastUsedAt),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}, nil
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
			AccountID:    row.AccountID,
			AccountName:  row.AccountName,
			RoleID:       row.RoleID,
			RoleName:     row.RoleName,
			RoleTypeCode: row.RoleTypeCode,
			LastUsedAt:   db.TimeFromNullTime(row.LastUsedAt),
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

	queryLike := accountUserQueryLike(params.Query)

	countResult, err := r.queries.CountAccountUsersFiltered(ctx, sqlc.CountAccountUsersFilteredParams{
		AccountID:      params.AccountID,
		IncludeRemoved: params.IncludeRemoved,
		RoleType:       db.NullStringPtr(params.RoleType),
		Query:          db.NullStringPtr(params.Query),
		Query_2:        db.NullStringPtr(params.Query),
		QueryLike:      queryLike,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	base := sqlc.ListAccountUsersForwardParams{
		AccountID:      params.AccountID,
		IncludeRemoved: params.IncludeRemoved,
		RoleType:       db.NullStringPtr(params.RoleType),
		Query:          db.NullStringPtr(params.Query),
		Query_2:        db.NullStringPtr(params.Query),
		QueryLike:      queryLike,
		Limit:          params.Limit + 1,
	}

	var cursorDir *pagination.Direction
	var items []*domain.AccountUserDetail
	var pageInfo pagination.PageInfo

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			brows, err := r.queries.ListAccountUsersBackward(ctx, sqlc.ListAccountUsersBackwardParams{
				AccountID:       base.AccountID,
				IncludeRemoved:  base.IncludeRemoved,
				RoleType:        base.RoleType,
				Query:           base.Query,
				Query_2:         base.Query_2,
				QueryLike:       base.QueryLike,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           base.Limit,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			details := make([]*domain.AccountUserDetail, len(brows))
			for i := range brows {
				details[i] = mapAccountUserDetailBackwardRow(brows[i])
			}
			items, pageInfo = pagination.BuildPageString(details, params.Limit, cursorDir, accountUserDetailCreatedAt, accountUserDetailID)
		} else {
			frows, err := r.queries.ListAccountUsersForward(ctx, sqlc.ListAccountUsersForwardParams{
				AccountID:       base.AccountID,
				IncludeRemoved:  base.IncludeRemoved,
				RoleType:        base.RoleType,
				Query:           base.Query,
				Query_2:         base.Query_2,
				QueryLike:       base.QueryLike,
				CursorCreatedAt: sql.NullTime{Time: cur.OccurredAt, Valid: true},
				CursorID:        sql.NullString{String: cur.ID, Valid: true},
				Limit:           base.Limit,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			details := make([]*domain.AccountUserDetail, len(frows))
			for i := range frows {
				details[i] = mapAccountUserDetailRow(frows[i])
			}
			items, pageInfo = pagination.BuildPageString(details, params.Limit, cursorDir, accountUserDetailCreatedAt, accountUserDetailID)
		}
	} else {
		frows, err := r.queries.ListAccountUsersForward(ctx, base)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		details := make([]*domain.AccountUserDetail, len(frows))
		for i := range frows {
			details[i] = mapAccountUserDetailRow(frows[i])
		}
		items, pageInfo = pagination.BuildPageString(details, params.Limit, cursorDir, accountUserDetailCreatedAt, accountUserDetailID)
	}

	return &domain.ListAccountUsersResult{
		Items:      items,
		PageInfo:   pageInfo,
		TotalCount: countResult,
	}, nil
}

func accountUserQueryLike(query *string) interface{} {
	if query == nil {
		return nil
	}
	if *query == "" {
		return nil
	}
	return *query
}

func accountUserDetailCreatedAt(d *domain.AccountUserDetail) time.Time { return d.CreatedAt }
func accountUserDetailID(d *domain.AccountUserDetail) string           { return d.ID }

func mapAccountUserDetailBackwardRow(row sqlc.ListAccountUsersBackwardRow) *domain.AccountUserDetail {
	return &domain.AccountUserDetail{
		ID:             row.ID,
		UserID:         row.UserID,
		Name:           db.StringFromNullString(row.Name),
		Email:          db.StringFromNullString(row.Email),
		Username:       db.StringFromNullString(row.Username),
		ImageURL:       db.StringFromNullString(row.ImageUrl),
		EmailVerified:  row.EmailVerified.Valid,
		RoleID:         db.StringFromNullString(row.RoleID),
		RoleName:       db.StringFromNullString(row.RoleName),
		RoleTypeCode:   db.StringFromNullString(row.RoleTypeCode),
		DepartmentID:   db.StringFromNullString(row.DepartmentID),
		DepartmentName: db.StringFromNullString(row.DepartmentName),
		StatusCode:     constants.AccountUserStatus(row.StatusCode),
		LastUsedAt:     db.TimeFromNullTime(row.LastUsedAt),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func (r *accountUserRepoImpl) GetDetail(ctx context.Context, accountID, userID string) (*domain.AccountUserDetail, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.get_detail")
	defer span.End()

	row, err := r.queries.GetAccountUserDetail(ctx, sqlc.GetAccountUserDetailParams{
		AccountID: accountID,
		UserID:    userID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapAccountUserDetailFromGetRow(row), nil
}

func (r *accountUserRepoImpl) GetDetailByAccountAndID(ctx context.Context, accountID, accountUserID string) (*domain.AccountUserDetail, *apierror.APIError) {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.get_detail_by_account_and_id")
	defer span.End()

	row, err := r.queries.GetAccountUserDetailByAccountAndID(ctx, sqlc.GetAccountUserDetailByAccountAndIDParams{
		AccountID: accountID,
		ID:        accountUserID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapAccountUserDetailFromGetByIDRow(row), nil
}

func (r *accountUserRepoImpl) Create(ctx context.Context, id, accountID, userID string, roleID, departmentID *string) *apierror.APIError {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.create")
	defer span.End()

	err := r.queries.InsertAccountUser(ctx, sqlc.InsertAccountUserParams{
		ID:           id,
		AccountID:    accountID,
		UserID:       userID,
		RoleID:       db.NullStringPtr(roleID),
		DepartmentID: db.NullStringPtr(departmentID),
		StatusCode:   "active",
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountUserRepoImpl) Update(ctx context.Context, accountUserID string, roleID, departmentID *string) *apierror.APIError {
	ctx, span := accountUserRepoTracer.Start(ctx, "repository.account_user.update")
	defer span.End()

	err := r.queries.UpdateAccountUserRoleAndDepartment(ctx, sqlc.UpdateAccountUserRoleAndDepartmentParams{
		ID:           accountUserID,
		RoleID:       db.NullStringPtr(roleID),
		DepartmentID: db.NullStringPtr(departmentID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
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

func mapAccountUserDetailRow(row sqlc.ListAccountUsersForwardRow) *domain.AccountUserDetail {
	return &domain.AccountUserDetail{
		ID:             row.ID,
		UserID:         row.UserID,
		Name:           db.StringFromNullString(row.Name),
		Email:          db.StringFromNullString(row.Email),
		Username:       db.StringFromNullString(row.Username),
		ImageURL:       db.StringFromNullString(row.ImageUrl),
		EmailVerified:  row.EmailVerified.Valid,
		RoleID:         db.StringFromNullString(row.RoleID),
		RoleName:       db.StringFromNullString(row.RoleName),
		RoleTypeCode:   db.StringFromNullString(row.RoleTypeCode),
		DepartmentID:   db.StringFromNullString(row.DepartmentID),
		DepartmentName: db.StringFromNullString(row.DepartmentName),
		StatusCode:     constants.AccountUserStatus(row.StatusCode),
		LastUsedAt:     db.TimeFromNullTime(row.LastUsedAt),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func mapAccountUserDetailFromGetRow(row sqlc.GetAccountUserDetailRow) *domain.AccountUserDetail {
	return &domain.AccountUserDetail{
		ID:             row.ID,
		UserID:         row.UserID,
		Name:           db.StringFromNullString(row.Name),
		Email:          db.StringFromNullString(row.Email),
		Username:       db.StringFromNullString(row.Username),
		ImageURL:       db.StringFromNullString(row.ImageUrl),
		EmailVerified:  row.EmailVerified.Valid,
		RoleID:         db.StringFromNullString(row.RoleID),
		RoleName:       db.StringFromNullString(row.RoleName),
		RoleTypeCode:   db.StringFromNullString(row.RoleTypeCode),
		DepartmentID:   db.StringFromNullString(row.DepartmentID),
		DepartmentName: db.StringFromNullString(row.DepartmentName),
		StatusCode:     constants.AccountUserStatus(row.StatusCode),
		LastUsedAt:     db.TimeFromNullTime(row.LastUsedAt),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func mapAccountUserDetailFromGetByIDRow(row sqlc.GetAccountUserDetailByAccountAndIDRow) *domain.AccountUserDetail {
	return &domain.AccountUserDetail{
		ID:             row.ID,
		UserID:         row.UserID,
		Name:           db.StringFromNullString(row.Name),
		Email:          db.StringFromNullString(row.Email),
		Username:       db.StringFromNullString(row.Username),
		ImageURL:       db.StringFromNullString(row.ImageUrl),
		EmailVerified:  row.EmailVerified.Valid,
		RoleID:         db.StringFromNullString(row.RoleID),
		RoleName:       db.StringFromNullString(row.RoleName),
		RoleTypeCode:   db.StringFromNullString(row.RoleTypeCode),
		DepartmentID:   db.StringFromNullString(row.DepartmentID),
		DepartmentName: db.StringFromNullString(row.DepartmentName),
		StatusCode:     constants.AccountUserStatus(row.StatusCode),
		LastUsedAt:     db.TimeFromNullTime(row.LastUsedAt),
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
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
			RoleTypeCode:             db.StringFromNullString(row.RoleTypeCode),
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
