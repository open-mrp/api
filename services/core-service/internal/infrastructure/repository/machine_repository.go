package repository

import (
	"context"
	gosql "database/sql"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/pagination"
	"github.com/open-mrp/apikit/tracing"
)

var machineRepoTracer = tracing.GetTracer("core-service.machine_repository")

type machineRepoImpl struct {
	queries *sqlc.Queries
}

func NewMachineRepo(queries *sqlc.Queries) domain.MachineRepo {
	return &machineRepoImpl{queries: queries}
}

func machineCreatedAt(m *domain.Machine) time.Time { return m.CreatedAt }
func machineID(m *domain.Machine) string           { return m.ID }

func machToNullString(s *string) gosql.NullString {
	if s == nil {
		return gosql.NullString{}
	}
	return gosql.NullString{String: *s, Valid: true}
}

// machBuildSearchParams matches machine names that start with the search term.
func machBuildSearchParams(query *string) gosql.NullString {
	if query == nil || *query == "" {
		return gosql.NullString{}
	}
	return gosql.NullString{String: db.EscapeLike(*query) + "%", Valid: true}
}

func mapGetMachineRow(row sqlc.GetMachineRow) *domain.Machine {
	m := &domain.Machine{
		ID:                  row.ID,
		Name:                row.Name,
		SerialNumber:        row.SerialNumber,
		DepartmentID:        &row.DepartmentID,
		DepartmentName:      &row.DepartmentName,
		DepartmentCreatedAt: &row.DepartmentCreatedAt,
		DepartmentUpdatedAt: &row.DepartmentUpdatedAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
	if row.Notes.Valid {
		m.Notes = &row.Notes.String
	}
	if row.ProductionStepID.Valid {
		m.ProductionStepID = &row.ProductionStepID.String
	}
	m.Status = constants.ArchiveStatusFromIsActive(row.IsActive)
	return m
}

func mapGetMachinesByIDsRow(row sqlc.GetMachinesByIDsRow) *domain.Machine {
	m := &domain.Machine{
		ID:                  row.ID,
		Name:                row.Name,
		SerialNumber:        row.SerialNumber,
		DepartmentID:        &row.DepartmentID,
		DepartmentName:      &row.DepartmentName,
		DepartmentCreatedAt: &row.DepartmentCreatedAt,
		DepartmentUpdatedAt: &row.DepartmentUpdatedAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
	if row.Notes.Valid {
		m.Notes = &row.Notes.String
	}
	if row.ProductionStepID.Valid {
		m.ProductionStepID = &row.ProductionStepID.String
	}
	m.Status = constants.ArchiveStatusFromIsActive(row.IsActive)
	return m
}

func mapFindMachinesBySerialNumbersRow(row sqlc.FindMachinesBySerialNumbersRow) *domain.Machine {
	m := &domain.Machine{
		ID:                  row.ID,
		Name:                row.Name,
		SerialNumber:        row.SerialNumber,
		DepartmentID:        &row.DepartmentID,
		DepartmentName:      &row.DepartmentName,
		DepartmentCreatedAt: &row.DepartmentCreatedAt,
		DepartmentUpdatedAt: &row.DepartmentUpdatedAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
	if row.Notes.Valid {
		m.Notes = &row.Notes.String
	}
	if row.ProductionStepID.Valid {
		m.ProductionStepID = &row.ProductionStepID.String
	}
	m.Status = constants.ArchiveStatusFromIsActive(row.IsActive)
	return m
}

func mapFindMachinesByNamesRow(row sqlc.FindMachinesByNamesRow) *domain.Machine {
	m := &domain.Machine{
		ID:                  row.ID,
		Name:                row.Name,
		SerialNumber:        row.SerialNumber,
		DepartmentID:        &row.DepartmentID,
		DepartmentName:      &row.DepartmentName,
		DepartmentCreatedAt: &row.DepartmentCreatedAt,
		DepartmentUpdatedAt: &row.DepartmentUpdatedAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
	if row.Notes.Valid {
		m.Notes = &row.Notes.String
	}
	if row.ProductionStepID.Valid {
		m.ProductionStepID = &row.ProductionStepID.String
	}
	m.Status = constants.ArchiveStatusFromIsActive(row.IsActive)
	return m
}

func (r *machineRepoImpl) GetByIDs(ctx context.Context, accountID string, ids []string) ([]*domain.Machine, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.get_by_ids")
	defer span.End()

	rows, err := r.queries.GetMachinesByIDs(ctx, sqlc.GetMachinesByIDsParams{
		Ids:       ids,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	machines := make([]*domain.Machine, len(rows))
	for i, row := range rows {
		machines[i] = mapGetMachinesByIDsRow(row)
	}
	return machines, nil
}

func (r *machineRepoImpl) SetProductionStep(ctx context.Context, accountID string, ids []string, productionStepID *string) *apierror.APIError {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.set_production_step")
	defer span.End()

	if len(ids) == 0 {
		return nil
	}
	err := r.queries.SetMachinesProductionStep(ctx, sqlc.SetMachinesProductionStepParams{
		ProductionStepID: ptrToNullString(productionStepID),
		Ids:              ids,
		AccountID:        accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *machineRepoImpl) List(ctx context.Context, params domain.ListMachinesParams) (*domain.ListMachinesResult, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.list")
	defer span.End()

	cursor, apiErr := decodeListCursor(params.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var search string
	if params.Query != nil {
		search = *params.Query
	}
	query, args := buildMachineListQuery(machineListFilter{
		accountID:     params.AccountID,
		search:        search,
		departmentIDs: params.DepartmentIDs,
		isActive:      params.IsActive,
		cursor:        cursor,
		limit:         params.Limit + 1,
	})
	rows, err := r.queries.DB().QueryContext(ctx, query, args...)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	defer func() { _ = rows.Close() }()
	scanned, err := scanMachineListRows(rows)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	machines := make([]*domain.Machine, len(scanned))
	tiers := make(map[string]int32, len(scanned))
	for i, row := range scanned {
		machines[i] = row.machine
		tiers[row.machine.ID] = row.tier
	}
	matchTier := func(m *domain.Machine) int32 { return tiers[m.ID] }
	result, pageInfo := pagination.BuildPageStringWithSearchRank(machines, params.Limit, cursorDirection(cursor), search != "",
		machineCreatedAt, machineID, matchTier)
	return &domain.ListMachinesResult{Machines: result, PageInfo: pageInfo}, nil
}

func (r *machineRepoImpl) Export(ctx context.Context, params domain.ExportMachinesParams) ([]*domain.Machine, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.export")
	defer span.End()

	rows, err := r.queries.ExportMachines(ctx, sqlc.ExportMachinesParams{
		AccountID:   params.AccountID,
		SearchQuery: machBuildSearchParams(params.Query),
		Limit:       exportQueryLimit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	machines := make([]*domain.Machine, len(rows))
	for i, row := range rows {
		machine := &domain.Machine{
			ID:           row.ID,
			Name:         row.Name,
			SerialNumber: row.SerialNumber,
			DepartmentID: &row.DepartmentID,
			CreatedAt:    row.CreatedAt,
			UpdatedAt:    row.UpdatedAt,
		}
		if row.Notes.Valid {
			machine.Notes = &row.Notes.String
		}
		if row.DepartmentName != "" {
			machine.DepartmentName = &row.DepartmentName
		}
		machines[i] = machine
	}
	return machines, nil
}

func (r *machineRepoImpl) Get(ctx context.Context, params domain.GetMachineParams) (*domain.Machine, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.get")
	defer span.End()

	row, err := r.queries.GetMachine(ctx, sqlc.GetMachineParams{
		ID:        params.MachineID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapGetMachineRow(row), nil
}

func (r *machineRepoImpl) Create(ctx context.Context, id string, params domain.CreateMachineParams) (*domain.Machine, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.create")
	defer span.End()

	err := r.queries.InsertMachine(ctx, sqlc.InsertMachineParams{
		ID:           id,
		AccountID:    params.AccountID,
		Name:         params.Name,
		SerialNumber: params.SerialNumber,
		Notes:        machToNullString(params.Notes),
		DepartmentID: params.DepartmentID,
		IsActive:     params.Status != constants.ArchiveStatusArchived,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetMachineParams{AccountID: params.AccountID, MachineID: id})
}

func (r *machineRepoImpl) Update(ctx context.Context, params domain.UpdateMachineParams) (*domain.Machine, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.update")
	defer span.End()

	result, err := r.queries.UpdateMachine(ctx, sqlc.UpdateMachineParams{
		ID:           params.MachineID,
		AccountID:    params.AccountID,
		Name:         machToNullString(params.Name),
		SerialNumber: machToNullString(params.SerialNumber),
		Notes:        field.StringToNullString(params.Notes),
		IsActive:     archiveStatusToNullBool(params.Status),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Machine not found."))
	}

	return r.Get(ctx, domain.GetMachineParams{AccountID: params.AccountID, MachineID: params.MachineID})
}

func (r *machineRepoImpl) Delete(ctx context.Context, params domain.DeleteMachineParams) *apierror.APIError {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.delete")
	defer span.End()

	result, err := r.queries.DeleteMachine(ctx, sqlc.DeleteMachineParams{
		ID:        params.MachineID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to check rows affected."))
	}
	if rowsAffected == 0 {
		return tracing.Trace(span, apierror.NewNotFoundError("Machine not found."))
	}

	return nil
}

func (r *machineRepoImpl) ExistsByName(ctx context.Context, accountID, name string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.exists_by_name")
	defer span.End()

	count, err := r.queries.CountMachinesByName(ctx, sqlc.CountMachinesByNameParams{
		Name:      name,
		AccountID: accountID,
		ExcludeID: machToNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *machineRepoImpl) ExistsBySerialNumber(ctx context.Context, accountID, serialNumber string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.exists_by_serial_number")
	defer span.End()

	count, err := r.queries.CountMachinesBySerialNumber(ctx, sqlc.CountMachinesBySerialNumberParams{
		SerialNumber: serialNumber,
		AccountID:    accountID,
		ExcludeID:    machToNullString(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return count > 0, nil
}

func (r *machineRepoImpl) FindBySerialNumbers(ctx context.Context, accountID string, serialNumbers []string) ([]*domain.Machine, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.find_by_serial_numbers")
	defer span.End()

	if len(serialNumbers) == 0 {
		return nil, nil
	}

	rows, err := r.queries.FindMachinesBySerialNumbers(ctx, sqlc.FindMachinesBySerialNumbersParams{
		SerialNumbers: serialNumbers,
		AccountID:     accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	machines := make([]*domain.Machine, len(rows))
	for i, row := range rows {
		machines[i] = mapFindMachinesBySerialNumbersRow(row)
	}
	return machines, nil
}

func (r *machineRepoImpl) FindByNames(ctx context.Context, accountID string, names []string) ([]*domain.Machine, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.find_by_names")
	defer span.End()

	if len(names) == 0 {
		return nil, nil
	}

	rows, err := r.queries.FindMachinesByNames(ctx, sqlc.FindMachinesByNamesParams{
		Names:     names,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	machines := make([]*domain.Machine, len(rows))
	for i, row := range rows {
		machines[i] = mapFindMachinesByNamesRow(row)
	}
	return machines, nil
}

func machineItemCreatedAt(m *domain.MachineItem) time.Time { return m.CreatedAt }
func machineItemID(m *domain.MachineItem) string           { return m.ItemID }

func (r *machineRepoImpl) ListItems(ctx context.Context, params domain.ListMachineItemsParams) (*domain.ListMachineItemsResult, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.list_items")
	defer span.End()

	search := machBuildSearchParams(params.Query)
	page := func(items []*domain.MachineItem, cursorDir *pagination.Direction) *domain.ListMachineItemsResult {
		result, pageInfo := pagination.BuildPageString(items, params.Limit, cursorDir, machineItemCreatedAt, machineItemID)
		return &domain.ListMachineItemsResult{Items: result, PageInfo: pageInfo}
	}

	var cur *pagination.StringCursor
	if params.Cursor != nil {
		decoded, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewParameterInvalidError("cursor", "Invalid pagination cursor."))
		}
		cur = &decoded
	}
	query, args := buildMachineItemListQuery(params, search, cur, params.Limit+1)
	items, err := r.listMachineItemPage(ctx, query, args)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if cur == nil {
		return page(items, nil), nil
	}
	return page(items, &cur.Direction), nil
}

func (r *machineRepoImpl) ReplaceItems(ctx context.Context, accountID, machineID string, itemIDs []string) *apierror.APIError {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.replace_items")
	defer span.End()

	if apiErr := r.DeleteItems(ctx, accountID, machineID); apiErr != nil {
		return apiErr
	}
	for _, itemID := range itemIDs {
		if err := r.queries.InsertMachineItem(ctx, sqlc.InsertMachineItemParams{
			AccountID: accountID,
			MachineID: machineID,
			ItemID:    itemID,
		}); err != nil {
			return tracing.Trace(span, db.MapSQLError(err))
		}
	}
	return nil
}

func (r *machineRepoImpl) DeleteItems(ctx context.Context, accountID, machineID string) *apierror.APIError {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.delete_items")
	defer span.End()

	if err := r.queries.DeleteMachineItems(ctx, sqlc.DeleteMachineItemsParams{
		AccountID: accountID,
		MachineID: machineID,
	}); err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}
	return nil
}

func (r *machineRepoImpl) ListItemsByMachineIDs(ctx context.Context, accountID string, machineIDs []string) ([]domain.MachineItemByMachine, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.list_items_by_machine_ids")
	defer span.End()

	if len(machineIDs) == 0 {
		return nil, nil
	}

	rows, err := r.queries.ListMachineItemsByMachineIDs(ctx, sqlc.ListMachineItemsByMachineIDsParams{
		AccountID:  accountID,
		MachineIds: machineIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]domain.MachineItemByMachine, len(rows))
	for i, row := range rows {
		out[i] = domain.MachineItemByMachine{
			MachineID: row.MachineID,
			ItemID:    row.ItemID,
			SKU:       row.Sku,
		}
	}
	return out, nil
}

func (r *machineRepoImpl) CountItemsForMachines(ctx context.Context, accountID string, machineIDs []string) (int64, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.count_items_for_machines")
	defer span.End()

	if len(machineIDs) == 0 {
		return 0, nil
	}

	count, err := r.queries.CountMachineItemsForMachines(ctx, sqlc.CountMachineItemsForMachinesParams{
		AccountID:  accountID,
		MachineIds: machineIDs,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return count, nil
}

func (r *machineRepoImpl) SuggestItemsFromScans(ctx context.Context, params domain.SuggestMachineItemsFromScansParams) ([]*domain.MachineItemScanSuggestion, *apierror.APIError) {
	ctx, span := machineRepoTracer.Start(ctx, "repository.machine.suggest_items_from_scans")
	defer span.End()

	rows, err := r.queries.SuggestMachineItemsFromScans(ctx, sqlc.SuggestMachineItemsFromScansParams{
		AccountID:   params.AccountID,
		MachineID:   params.MachineID,
		WindowStart: gosql.NullTime{Time: params.WindowStart, Valid: true},
		WindowEnd:   gosql.NullTime{Time: params.WindowEnd, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	out := make([]*domain.MachineItemScanSuggestion, len(rows))
	for i, row := range rows {
		s := &domain.MachineItemScanSuggestion{
			ItemID:     row.ItemID,
			SKU:        row.Sku,
			BatchCount: row.BatchCount,
		}
		if t, ok := row.LastScannedAt.(time.Time); ok {
			s.LastScannedAt = t
		}
		if row.Description.Valid {
			s.Description = &row.Description.String
		}
		out[i] = s
	}
	return out, nil
}
