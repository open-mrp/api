package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"time"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/textutil"
	"github.com/augno/api/shared/tracing"
)

var accountGroupRepoTracer = tracing.GetTracer("core-service.account_group_repository")

var accountGroupDuplicateKeyMapping = db.DuplicateKeyMapping{
	"account_group_owner_account_id_name_key": func() *apierror.APIError {
		return apierror.NewConflictErrorWithParam("An account group with this name already exists.", "name")
	},
}

type accountGroupRepoImpl struct {
	queries *sqlc.Queries
}

func NewAccountGroupRepo(queries *sqlc.Queries) domain.AccountGroupRepo {
	return &accountGroupRepoImpl{queries: queries}
}

func accountGroupCreatedAt(ag *domain.AccountGroup) time.Time { return ag.CreatedAt }
func accountGroupID(ag *domain.AccountGroup) string           { return ag.ID }

func mapAccountGroupRow(
	id, ownerAccountID, name string,
	description gosql.NullString,
	commissionStatusCode, freightStatusCode, accountGroupTypeCode string,
	registrationFlowID gosql.NullString,
	defaultLeadTimeDays gosql.NullInt32,
	createdAt, updatedAt time.Time,
) *domain.AccountGroup {
	ag := &domain.AccountGroup{
		ID:                   id,
		OwnerAccountID:       ownerAccountID,
		Name:                 name,
		CommissionPolicyCode: commissionStatusCode,
		FreightPolicyCode:    freightStatusCode,
		AccountGroupTypeCode: accountGroupTypeCode,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
	}
	if description.Valid {
		ag.Description = &description.String
	}
	if registrationFlowID.Valid {
		ag.RegistrationFlowID = &registrationFlowID.String
	}
	if defaultLeadTimeDays.Valid {
		ag.DefaultLeadTimeDays = &defaultLeadTimeDays.Int32
	}
	return ag
}

func mapForwardAccountGroupRow(row sqlc.ListAccountGroupsForwardRow) *domain.AccountGroup {
	return mapAccountGroupRow(
		row.ID, row.OwnerAccountID, row.Name, row.Description,
		row.CommissionStatusCode, row.FreightStatusCode, row.AccountGroupTypeCode,
		row.RegistrationFlowID, row.DefaultLeadTimeDays, row.CreatedAt, row.UpdatedAt,
	)
}

func mapBackwardAccountGroupRow(row sqlc.ListAccountGroupsBackwardRow) *domain.AccountGroup {
	return mapAccountGroupRow(
		row.ID, row.OwnerAccountID, row.Name, row.Description,
		row.CommissionStatusCode, row.FreightStatusCode, row.AccountGroupTypeCode,
		row.RegistrationFlowID, row.DefaultLeadTimeDays, row.CreatedAt, row.UpdatedAt,
	)
}

func mapGetAccountGroupRow(row sqlc.GetAccountGroupRow) *domain.AccountGroup {
	return mapAccountGroupRow(
		row.ID, row.OwnerAccountID, row.Name, row.Description,
		row.CommissionStatusCode, row.FreightStatusCode, row.AccountGroupTypeCode,
		row.RegistrationFlowID, row.DefaultLeadTimeDays, row.CreatedAt, row.UpdatedAt,
	)
}

func mapGetAccountGroupsByIDsRow(row sqlc.GetAccountGroupsByIDsRow) *domain.AccountGroup {
	return mapAccountGroupRow(
		row.ID, row.OwnerAccountID, row.Name, row.Description,
		row.CommissionStatusCode, row.FreightStatusCode, row.AccountGroupTypeCode,
		row.RegistrationFlowID, row.DefaultLeadTimeDays, row.CreatedAt, row.UpdatedAt,
	)
}

func (r *accountGroupRepoImpl) List(ctx context.Context, params domain.ListAccountGroupsParams) (*domain.ListAccountGroupsResult, *apierror.APIError) {
	ctx, span := accountGroupRepoTracer.Start(ctx, "repository.account_group.list")
	defer span.End()

	searchQuery := db.NullStringLikePtr(params.Query)
	typeFilter := db.NullStringPtr(params.Type)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListAccountGroupsBackward(ctx, sqlc.ListAccountGroupsBackwardParams{
				OwnerAccountID:  params.AccountID,
				SearchQuery:     searchQuery,
				TypeFilter:      typeFilter,
				CursorCreatedAt: cur.OccurredAt,
				CursorID:        cur.ID,
				Limit:           params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			groups := make([]*domain.AccountGroup, len(rows))
			for i, row := range rows {
				groups[i] = mapBackwardAccountGroupRow(row)
			}
			result, pageInfo := pagination.BuildPageString(groups, params.Limit, cursorDir, accountGroupCreatedAt, accountGroupID)
			return &domain.ListAccountGroupsResult{AccountGroups: result, PageInfo: pageInfo}, nil
		}

		// Forward
		rows, err := r.queries.ListAccountGroupsForward(ctx, sqlc.ListAccountGroupsForwardParams{
			OwnerAccountID:  params.AccountID,
			SearchQuery:     searchQuery,
			TypeFilter:      typeFilter,
			CursorCreatedAt: gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:        gosql.NullString{String: cur.ID, Valid: true},
			Limit:           params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		groups := make([]*domain.AccountGroup, len(rows))
		for i, row := range rows {
			groups[i] = mapForwardAccountGroupRow(row)
		}
		result, pageInfo := pagination.BuildPageString(groups, params.Limit, cursorDir, accountGroupCreatedAt, accountGroupID)
		return &domain.ListAccountGroupsResult{AccountGroups: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListAccountGroupsForward(ctx, sqlc.ListAccountGroupsForwardParams{
		OwnerAccountID: params.AccountID,
		SearchQuery:    searchQuery,
		TypeFilter:     typeFilter,
		Limit:          params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	groups := make([]*domain.AccountGroup, len(rows))
	for i, row := range rows {
		groups[i] = mapForwardAccountGroupRow(row)
	}
	result, pageInfo := pagination.BuildPageString(groups, params.Limit, cursorDir, accountGroupCreatedAt, accountGroupID)
	return &domain.ListAccountGroupsResult{AccountGroups: result, PageInfo: pageInfo}, nil
}

func (r *accountGroupRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.AccountGroup, *apierror.APIError) {
	ctx, span := accountGroupRepoTracer.Start(ctx, "repository.account_group.get_by_ids")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.queries.GetAccountGroupsByIDs(ctx, sqlc.GetAccountGroupsByIDsParams{
		Ids:            ids,
		OwnerAccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.AccountGroup, len(rows))
	for i, row := range rows {
		out[i] = mapGetAccountGroupsByIDsRow(row)
	}
	return out, nil
}

func (r *accountGroupRepoImpl) Get(ctx context.Context, accountID, id string) (*domain.AccountGroup, *apierror.APIError) {
	ctx, span := accountGroupRepoTracer.Start(ctx, "repository.account_group.get")
	defer span.End()

	row, err := r.queries.GetAccountGroup(ctx, sqlc.GetAccountGroupParams{
		ID:             id,
		OwnerAccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetAccountGroupRow(row), nil
}

func (r *accountGroupRepoImpl) Create(ctx context.Context, id string, params domain.CreateAccountGroupParams) (*domain.AccountGroup, *apierror.APIError) {
	ctx, span := accountGroupRepoTracer.Start(ctx, "repository.account_group.create")
	defer span.End()

	err := r.queries.InsertAccountGroup(ctx, sqlc.InsertAccountGroupParams{
		ID:                   id,
		OwnerAccountID:       params.AccountID,
		Name:                 params.Name,
		Description:          toNullString(params.Description),
		CommissionStatusCode: params.CommissionPolicyCode,
		FreightStatusCode:    params.FreightPolicyCode,
		AccountGroupTypeCode: params.AccountGroupTypeCode,
		DefaultLeadTimeDays:  toNullInt32(params.DefaultLeadTimeDays),
	})
	if apiErr := db.MapSQLErrorWithDuplicateKeys(err, accountGroupDuplicateKeyMapping); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, params.AccountID, id)
}

func (r *accountGroupRepoImpl) Update(ctx context.Context, params domain.UpdateAccountGroupParams) (*domain.AccountGroup, *apierror.APIError) {
	ctx, span := accountGroupRepoTracer.Start(ctx, "repository.account_group.update")
	defer span.End()

	result, err := r.queries.UpdateAccountGroup(ctx, sqlc.UpdateAccountGroupParams{
		ID:                   params.AccountGroupID,
		OwnerAccountID:       params.AccountID,
		Name:                 toNullString(params.Name),
		Description:          field.StringToNullString(params.Description),
		CommissionStatusCode: toNullString(params.CommissionPolicyCode),
		FreightStatusCode:    toNullString(params.FreightPolicyCode),
		DefaultLeadTimeDays:  field.Int32ToNullInt32(params.DefaultLeadTimeDays),
	})
	if apiErr := db.MapSQLErrorWithDuplicateKeys(err, accountGroupDuplicateKeyMapping); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Account group not found."))
	}

	return r.Get(ctx, params.AccountID, params.AccountGroupID)
}

func (r *accountGroupRepoImpl) Delete(ctx context.Context, params domain.DeleteAccountGroupParams) *apierror.APIError {
	ctx, span := accountGroupRepoTracer.Start(ctx, "repository.account_group.delete")
	defer span.End()

	result, err := r.queries.DeleteAccountGroup(ctx, sqlc.DeleteAccountGroupParams{
		ID:             params.AccountGroupID,
		OwnerAccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Account group not found."))
	}

	return nil
}

func (r *accountGroupRepoImpl) CheckAccountGroupNotInUse(ctx context.Context, accountGroup *domain.AccountGroup) *apierror.APIError {
	ctx, span := accountGroupRepoTracer.Start(ctx, "repository.account_group.check_not_in_use")
	defer span.End()

	customerCount, err := r.queries.CountAccountGroupUsageInAccountRelation(ctx, sqlc.CountAccountGroupUsageInAccountRelationParams{
		OwnerAccountID: accountGroup.OwnerAccountID,
		AccountGroupID: toNullString(&accountGroup.ID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check account group usage in customer records."))
	}
	if accountGroup.AccountGroupTypeCode == string(constants.AccountGroupTypeTypeGroup) && customerCount > 0 {
		return apierror.NewValidationError(
			fmt.Sprintf(
				"Cannot delete account group '%s'. %d customer %s %s on it as %s customer type.",
				accountGroup.Name,
				customerCount,
				textutil.Pluralize(int(customerCount), "record", "records"),
				textutil.Pluralize(int(customerCount), "relies", "rely"),
				textutil.Pluralize(int(customerCount), "its", "their"),
			),
		)
	}

	productLineCount, err := r.queries.CountAccountGroupUsageInAccountGroupProductLine(ctx, accountGroup.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check account group usage in account group product lines."))
	}
	if productLineCount > 0 {
		return apierror.NewValidationError(
			fmt.Sprintf(
				"Cannot delete account group '%s'. %d product line access %s %s on it.",
				accountGroup.Name,
				productLineCount,
				textutil.Pluralize(int(productLineCount), "record", "records"),
				textutil.Pluralize(int(productLineCount), "relies", "rely"),
			),
		)
	}

	quantityDiscountCount, err := r.queries.CountAccountGroupUsageInAccountGroupQuantityDiscount(ctx, accountGroup.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check account group usage in account group quantity discounts."))
	}
	if quantityDiscountCount > 0 {
		return apierror.NewValidationError(
			fmt.Sprintf(
				"Cannot delete account group '%s'. %d volume discount %s %s on it.",
				accountGroup.Name,
				quantityDiscountCount,
				textutil.Pluralize(int(quantityDiscountCount), "assignment", "assignments"),
				textutil.Pluralize(int(quantityDiscountCount), "relies", "rely"),
			),
		)
	}

	pricingAssignmentCount, err := r.queries.CountAccountGroupUsageInAccountRelationPriceGroup(ctx, accountGroup.ID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check account group usage in account relation price groups."))
	}
	if accountGroup.AccountGroupTypeCode != string(constants.AccountGroupTypePricingGroup) && pricingAssignmentCount > 0 {
		return apierror.NewValidationError(
			fmt.Sprintf(
				"Cannot delete account group '%s'. %d pricing %s %s on it.",
				accountGroup.Name,
				pricingAssignmentCount,
				textutil.Pluralize(int(pricingAssignmentCount), "assignment", "assignments"),
				textutil.Pluralize(int(pricingAssignmentCount), "relies", "rely"),
			),
		)
	}

	registrationFlowCount, err := r.queries.CountAccountGroupUsageInRegistrationFlow(ctx, sqlc.CountAccountGroupUsageInRegistrationFlowParams{
		AccountGroupID: accountGroup.ID,
		OwnerAccountID: accountGroup.OwnerAccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check account group usage in registration flows."))
	}
	if registrationFlowCount > 0 {
		return apierror.NewValidationError(
			fmt.Sprintf("Cannot delete account group '%s'. It is connected to an active registration flow.", accountGroup.Name),
		)
	}

	return nil
}

func (r *accountGroupRepoImpl) DeleteAccountRelationPriceGroupsByAccountGroupID(ctx context.Context, accountGroupID string) *apierror.APIError {
	ctx, span := accountGroupRepoTracer.Start(ctx, "repository.account_group.delete_account_relation_price_groups_by_account_group_id")
	defer span.End()

	if apiErr := db.MapSQLError(r.queries.DeleteAccountRelationPriceGroupsByAccountGroupID(ctx, accountGroupID)); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *accountGroupRepoImpl) ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := accountGroupRepoTracer.Start(ctx, "repository.account_group.exists_by_name")
	defer span.End()

	count, err := r.queries.CountAccountGroupsByName(ctx, sqlc.CountAccountGroupsByNameParams{
		Name:           name,
		OwnerAccountID: accountID,
		ExcludeID:      toNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}
