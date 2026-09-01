package repository

import (
	"context"
	gosql "database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
)

var productionRunRepoTracer = tracing.GetTracer("core-service.production_run_repository")

type productionRunRepoImpl struct {
	queries *sqlc.Queries
}

func NewProductionRunRepo(queries *sqlc.Queries) domain.ProductionRunRepo {
	return &productionRunRepoImpl{queries: queries}
}

func productionRunSummaryCreatedAt(d *domain.ProductionRunSummary) time.Time { return d.CreatedAt }
func productionRunSummaryID(d *domain.ProductionRunSummary) string           { return d.ID }

func buildProductionRunSearchParams(query *string) (numberQuery gosql.NullString, batchIDQuery gosql.NullString) {
	if query == nil || *query == "" {
		return gosql.NullString{}, gosql.NullString{}
	}
	return gosql.NullString{String: "%" + db.EscapeLike(*query) + "%", Valid: true},
		gosql.NullString{String: *query + "%", Valid: true}
}

func buildProductionRunListFilters(params domain.ListProductionRunsParams) (
	includeStatusFilter bool, statusOpen bool, statusClosed bool,
	includeItemFilter bool, itemIDs []string,
	includeMachineFilter bool, machineIDs []string,
) {
	if params.Status != nil {
		includeStatusFilter = true
		switch *params.Status {
		case "open":
			statusOpen = true
		case "closed":
			statusClosed = true
		}
	}

	includeItemFilter = len(params.ItemIDs) > 0
	itemIDs = params.ItemIDs

	includeMachineFilter = len(params.MachineIDs) > 0
	machineIDs = params.MachineIDs

	return
}

func (r *productionRunRepoImpl) Export(ctx context.Context, params domain.ExportProductionRunsParams) ([]*domain.ProductionRunExport, *apierror.APIError) {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.export")
	defer span.End()

	searchQuery, _ := buildProductionRunSearchParams(params.Query)
	rows, err := r.queries.ExportProductionRuns(ctx, sqlc.ExportProductionRunsParams{
		AccountID:   params.AccountID,
		SearchQuery: searchQuery,
		Limit:       exportQueryLimit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	runs := make([]*domain.ProductionRunExport, len(rows))
	byID := make(map[string]*domain.ProductionRunExport, len(rows))
	ids := make([]string, len(rows))
	for i, row := range rows {
		run := &domain.ProductionRunExport{
			ID:                  row.ID,
			Number:              row.Number,
			ResponsibleUserName: row.ResponsibleUserName,
		}
		if row.StartedAt.Valid {
			run.StartedAt = &row.StartedAt.Time
		}
		if row.CompletedAt.Valid {
			run.CompletedAt = &row.CompletedAt.Time
		}
		if row.OrderID.Valid {
			run.OrderID = &row.OrderID.String
		}
		runs[i] = run
		byID[row.ID] = run
		ids[i] = row.ID
	}

	if len(ids) > 0 {
		if apiErr := r.attachExportBatches(ctx, params.AccountID, ids, byID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return runs, nil
}

// loads the runs' batches and those batches' machines, two queries regardless of
// how many runs or batches there are
func (r *productionRunRepoImpl) attachExportBatches(ctx context.Context, accountID string, runIDs []string, byID map[string]*domain.ProductionRunExport) *apierror.APIError {
	// production_run_id is nullable on batch, so the IN list is too.
	nullableRunIDs := make([]gosql.NullString, len(runIDs))
	for i, id := range runIDs {
		nullableRunIDs[i] = gosql.NullString{String: id, Valid: true}
	}

	batchRows, err := r.queries.ExportProductionRunBatches(ctx, sqlc.ExportProductionRunBatchesParams{
		ProductionRunIds: nullableRunIDs,
		AccountID:        accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	if len(batchRows) == 0 {
		return nil
	}

	batchIDs := make([]string, len(batchRows))
	for i, row := range batchRows {
		batchIDs[i] = row.ID
	}

	machineRows, err := r.queries.ExportBatchMachines(ctx, batchIDs)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	machinesByBatch := make(map[string][]string, len(batchIDs))
	for _, mr := range machineRows {
		machinesByBatch[mr.BatchID] = append(machinesByBatch[mr.BatchID], mr.Name)
	}

	for _, row := range batchRows {
		if !row.ProductionRunID.Valid {
			continue
		}
		run, ok := byID[row.ProductionRunID.String]
		if !ok {
			continue
		}
		batch := domain.ProductionRunExportBatch{
			ID:            row.ID,
			ItemSKU:       row.ItemSku,
			QuantityValue: row.QuantityValue,
			QuantityUnit:  row.QuantityUnitAbbreviation,
			MachineNames:  machinesByBatch[row.ID],
		}
		if row.DepartmentName.Valid {
			batch.DepartmentName = &row.DepartmentName.String
		}
		if row.ScannedAt.Valid {
			batch.ScannedAt = &row.ScannedAt.Time
		}
		run.Batches = append(run.Batches, batch)
	}

	return nil
}

func (r *productionRunRepoImpl) List(ctx context.Context, params domain.ListProductionRunsParams) (*domain.ListProductionRunsResult, *apierror.APIError) {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.list")
	defer span.End()

	searchQuery, batchIDQuery := buildProductionRunSearchParams(params.Query)
	startDate := parseDateString(params.StartDate)
	endDate := parseDateString(params.EndDate)

	includeStatusFilter, statusOpen, statusClosed,
		includeItemFilter, itemIDs,
		includeMachineFilter, machineIDs := buildProductionRunListFilters(params)

	var cursorDir *pagination.Direction

	if params.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.queries.ListProductionRunsBackward(ctx, sqlc.ListProductionRunsBackwardParams{
				AccountID:            params.AccountID,
				SearchQuery:          searchQuery,
				BatchIDQuery:         batchIDQuery,
				IncludeStatusFilter:  includeStatusFilter,
				StatusOpen:           statusOpen,
				StatusClosed:         statusClosed,
				IncludeItemFilter:    includeItemFilter,
				ItemIds:              itemIDs,
				IncludeMachineFilter: includeMachineFilter,
				MachineIds:           machineIDs,
				StartDate:            startDate,
				EndDate:              endDate,
				CursorCreatedAt:      gosql.NullTime{Time: cur.OccurredAt, Valid: true},
				CursorID:             gosql.NullString{String: cur.ID, Valid: true},
				Limit:                params.Limit + 1,
			})
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			runs := make([]*domain.ProductionRunSummary, len(rows))
			for i, row := range rows {
				runs[i] = mapBackwardProductionRunRow(row)
			}
			result, pageInfo := pagination.BuildPageString(runs, params.Limit, cursorDir, productionRunSummaryCreatedAt, productionRunSummaryID)
			return &domain.ListProductionRunsResult{ProductionRuns: result, PageInfo: pageInfo}, nil
		}

		// Forward with cursor
		rows, err := r.queries.ListProductionRunsForward(ctx, sqlc.ListProductionRunsForwardParams{
			AccountID:            params.AccountID,
			SearchQuery:          searchQuery,
			IncludeStatusFilter:  includeStatusFilter,
			StatusOpen:           statusOpen,
			StatusClosed:         statusClosed,
			IncludeItemFilter:    includeItemFilter,
			ItemIds:              itemIDs,
			IncludeMachineFilter: includeMachineFilter,
			MachineIds:           machineIDs,
			StartDate:            startDate,
			EndDate:              endDate,
			CursorCreatedAt:      gosql.NullTime{Time: cur.OccurredAt, Valid: true},
			CursorID:             gosql.NullString{String: cur.ID, Valid: true},
			Limit:                params.Limit + 1,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		runs := make([]*domain.ProductionRunSummary, len(rows))
		for i, row := range rows {
			runs[i] = mapForwardProductionRunRow(row)
		}
		result, pageInfo := pagination.BuildPageString(runs, params.Limit, cursorDir, productionRunSummaryCreatedAt, productionRunSummaryID)
		return &domain.ListProductionRunsResult{ProductionRuns: result, PageInfo: pageInfo}, nil
	}

	// No cursor — first page
	rows, err := r.queries.ListProductionRunsForward(ctx, sqlc.ListProductionRunsForwardParams{
		AccountID:            params.AccountID,
		SearchQuery:          searchQuery,
		IncludeStatusFilter:  includeStatusFilter,
		StatusOpen:           statusOpen,
		StatusClosed:         statusClosed,
		IncludeItemFilter:    includeItemFilter,
		ItemIds:              itemIDs,
		IncludeMachineFilter: includeMachineFilter,
		MachineIds:           machineIDs,
		StartDate:            startDate,
		EndDate:              endDate,
		CursorCreatedAt:      gosql.NullTime{},
		CursorID:             gosql.NullString{},
		Limit:                params.Limit + 1,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	runs := make([]*domain.ProductionRunSummary, len(rows))
	for i, row := range rows {
		runs[i] = mapForwardProductionRunRow(row)
	}
	result, pageInfo := pagination.BuildPageString(runs, params.Limit, cursorDir, productionRunSummaryCreatedAt, productionRunSummaryID)
	return &domain.ListProductionRunsResult{ProductionRuns: result, PageInfo: pageInfo}, nil
}

// resolvedResponsibleUserID prefers the account_user id resolved by the query; legacy rows store a user id in responsible_user_id and may have no account_user match, in which case the raw value is kept.
func resolvedResponsibleUserID(accountUserID gosql.NullString, raw string) string {
	if accountUserID.Valid {
		return accountUserID.String
	}
	return raw
}

func mapForwardProductionRunRow(row sqlc.ListProductionRunsForwardRow) *domain.ProductionRunSummary {
	s := &domain.ProductionRunSummary{
		ID:                row.ID,
		Number:            row.Number,
		ResponsibleUserID: resolvedResponsibleUserID(row.ResponsibleAccountUserID, row.ResponsibleUserID),
		BatchCount:        safeconv.Int64ToInt32(row.BatchCount),
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
	if row.ResponsibleUserName != "" {
		s.ResponsibleUserName = &row.ResponsibleUserName
	}
	if row.ResponsibleUserStatusCode.Valid {
		s.ResponsibleUserStatusCode = &row.ResponsibleUserStatusCode.String
	}
	if row.ResponsibleUserCreatedAt.Valid {
		s.ResponsibleUserCreatedAt = &row.ResponsibleUserCreatedAt.Time
	}
	if row.ResponsibleUserUpdatedAt.Valid {
		s.ResponsibleUserUpdatedAt = &row.ResponsibleUserUpdatedAt.Time
	}
	if row.StartedAt.Valid {
		s.StartedAt = &row.StartedAt.Time
	}
	if row.CompletedAt.Valid {
		s.CompletedAt = &row.CompletedAt.Time
	}
	return s
}

func mapBackwardProductionRunRow(row sqlc.ListProductionRunsBackwardRow) *domain.ProductionRunSummary {
	s := &domain.ProductionRunSummary{
		ID:                row.ID,
		Number:            row.Number,
		ResponsibleUserID: resolvedResponsibleUserID(row.ResponsibleAccountUserID, row.ResponsibleUserID),
		BatchCount:        safeconv.Int64ToInt32(row.BatchCount),
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
	if row.ResponsibleUserName != "" {
		s.ResponsibleUserName = &row.ResponsibleUserName
	}
	if row.ResponsibleUserStatusCode.Valid {
		s.ResponsibleUserStatusCode = &row.ResponsibleUserStatusCode.String
	}
	if row.ResponsibleUserCreatedAt.Valid {
		s.ResponsibleUserCreatedAt = &row.ResponsibleUserCreatedAt.Time
	}
	if row.ResponsibleUserUpdatedAt.Valid {
		s.ResponsibleUserUpdatedAt = &row.ResponsibleUserUpdatedAt.Time
	}
	if row.StartedAt.Valid {
		s.StartedAt = &row.StartedAt.Time
	}
	if row.CompletedAt.Valid {
		s.CompletedAt = &row.CompletedAt.Time
	}
	return s
}

func (r *productionRunRepoImpl) Get(ctx context.Context, params domain.GetProductionRunParams) (*domain.ProductionRun, *apierror.APIError) {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.get")
	defer span.End()

	row, err := r.queries.GetProductionRun(ctx, sqlc.GetProductionRunParams{
		ID:        params.ProductionRunID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	run := &domain.ProductionRun{
		ID:                row.ID,
		Number:            row.Number,
		ResponsibleUserID: resolvedResponsibleUserID(row.ResponsibleAccountUserID, row.ResponsibleUserID),
		AccountID:         row.AccountID,
		BatchCount:        safeconv.Int64ToInt32(row.BatchCount),
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
	if row.ResponsibleUserName != "" {
		run.ResponsibleUserName = &row.ResponsibleUserName
	}
	if row.ResponsibleUserStatusCode.Valid {
		run.ResponsibleUserStatusCode = &row.ResponsibleUserStatusCode.String
	}
	if row.ResponsibleUserCreatedAt.Valid {
		run.ResponsibleUserCreatedAt = &row.ResponsibleUserCreatedAt.Time
	}
	if row.ResponsibleUserUpdatedAt.Valid {
		run.ResponsibleUserUpdatedAt = &row.ResponsibleUserUpdatedAt.Time
	}
	if row.StartedAt.Valid {
		run.StartedAt = &row.StartedAt.Time
	}
	if row.CompletedAt.Valid {
		run.CompletedAt = &row.CompletedAt.Time
	}

	return run, nil
}

func (r *productionRunRepoImpl) Create(ctx context.Context, id string, params domain.CreateProductionRunParams, number string) (*domain.ProductionRun, *apierror.APIError) {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.create")
	defer span.End()

	err := r.queries.InsertProductionRun(ctx, sqlc.InsertProductionRunParams{
		ID:                id,
		ResponsibleUserID: params.ResponsibleUserID,
		Number:            number,
		AccountID:         params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return r.Get(ctx, domain.GetProductionRunParams{
		ProductionRunID: id,
		AccountID:       params.AccountID,
	})
}

func (r *productionRunRepoImpl) Update(ctx context.Context, params domain.UpdateProductionRunParams) (*domain.ProductionRun, *apierror.APIError) {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.update")
	defer span.End()

	if params.Number != nil {
		err := r.queries.UpdateProductionRunNumber(ctx, sqlc.UpdateProductionRunNumberParams{
			Number:    *params.Number,
			ID:        params.ProductionRunID,
			AccountID: params.AccountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	if params.ResponsibleUserID != nil {
		err := r.queries.UpdateProductionRunResponsibleUser(ctx, sqlc.UpdateProductionRunResponsibleUserParams{
			ResponsibleUserID: *params.ResponsibleUserID,
			ID:                params.ProductionRunID,
			AccountID:         params.AccountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	return r.Get(ctx, domain.GetProductionRunParams{
		ProductionRunID: params.ProductionRunID,
		AccountID:       params.AccountID,
	})
}

func (r *productionRunRepoImpl) Delete(ctx context.Context, params domain.DeleteProductionRunParams) *apierror.APIError {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.delete")
	defer span.End()

	err := r.queries.DeleteProductionRunByID(ctx, sqlc.DeleteProductionRunByIDParams{
		ID:        params.ProductionRunID,
		AccountID: params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionRunRepoImpl) ExistsByNumber(ctx context.Context, accountID, number string, excludeID *string) (bool, *apierror.APIError) {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.exists_by_number")
	defer span.End()

	count, err := r.queries.CountProductionRunsByNumber(ctx, sqlc.CountProductionRunsByNumberParams{
		AccountID: accountID,
		Number:    number,
		ExcludeID: db.NullStringPtr(excludeID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return count > 0, nil
}

func (r *productionRunRepoImpl) GetNextNumber(ctx context.Context, accountID string) (string, *apierror.APIError) {
	numbers, apiErr := r.GetNextNumbers(ctx, accountID, 1)
	if apiErr != nil {
		return "", apiErr
	}
	return numbers[0], nil
}

func (r *productionRunRepoImpl) GetNextNumbers(ctx context.Context, accountID string, count int) ([]string, *apierror.APIError) {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.get_next_numbers")
	defer span.End()

	if count <= 0 {
		return nil, nil
	}

	// Atomic rather than MAX(number)+1: two runs created at once — which releasing two
	// weeks back to back does — read the same maximum and collide on the unique number.
	seedID, apiErr := id.GenID(id.SysPropertyIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if err := r.queries.SeedProductionRunNumberCounter(ctx, sqlc.SeedProductionRunNumberCounterParams{
		ID:        seedID,
		AccountID: accountID,
	}); err != nil {
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	numbers := make([]string, 0, count)
	for range count {
		allocID, apiErr := id.GenID(id.SysPropertyIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		result, err := r.queries.AllocateNextProductionRunNumber(ctx, sqlc.AllocateNextProductionRunNumberParams{
			ID:        allocID,
			AccountID: accountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		nextNum, err := result.LastInsertId()
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Could not read the allocated production run number."))
		}
		numbers = append(numbers, fmt.Sprintf("%d", nextNum))
	}
	return numbers, nil
}

func (r *productionRunRepoImpl) IsCompleted(ctx context.Context, accountID, id string) (bool, *apierror.APIError) {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.is_completed")
	defer span.End()

	isCompleted, err := r.queries.IsProductionRunCompleted(ctx, sqlc.IsProductionRunCompletedParams{
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}

	return isCompleted != 0, nil
}

func (r *productionRunRepoImpl) DeleteBatchesByRun(ctx context.Context, accountID, productionRunID string) *apierror.APIError {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.delete_batches_by_run")
	defer span.End()

	err := r.queries.DeleteBatchesByProductionRunID(ctx, sqlc.DeleteBatchesByProductionRunIDParams{
		ProductionRunID: gosql.NullString{String: productionRunID, Valid: true},
		AccountID:       accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionRunRepoImpl) FindOrderIDsByRun(ctx context.Context, accountID, productionRunID string) ([]string, *apierror.APIError) {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.find_order_ids_by_run")
	defer span.End()

	ids, err := r.queries.FindSalesOrderIDsByProductionRunID(ctx, sqlc.FindSalesOrderIDsByProductionRunIDParams{
		ProductionRunID: gosql.NullString{String: productionRunID, Valid: true},
		AccountID:       accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return ids, nil
}

func (r *productionRunRepoImpl) UnlinkOrdersFromRun(ctx context.Context, accountID, productionRunID string) *apierror.APIError {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.unlink_orders_from_run")
	defer span.End()

	err := r.queries.UnlinkSalesOrdersFromProductionRun(ctx, sqlc.UnlinkSalesOrdersFromProductionRunParams{
		ProductionRunID: gosql.NullString{String: productionRunID, Valid: true},
		AccountID:       accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionRunRepoImpl) SetBatchProductionRunID(ctx context.Context, accountID, batchID, productionRunID string) *apierror.APIError {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.set_batch_production_run_id")
	defer span.End()

	err := r.queries.SetBatchProductionRunID(ctx, sqlc.SetBatchProductionRunIDParams{
		ProductionRunID: gosql.NullString{String: productionRunID, Valid: true},
		ID:              batchID,
		AccountID:       accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *productionRunRepoImpl) ListBatchesByRun(ctx context.Context, params domain.ListBatchesByProductionRunParams) (*domain.ListBatchesByProductionRunResult, *apierror.APIError) {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.list_batches_by_run")
	defer span.End()

	// Step 1: Get initial batch IDs in the production run.
	initialRows, err := r.queries.GetBatchIDsByProductionRun(ctx, sqlc.GetBatchIDsByProductionRunParams{
		ProductionRunID: gosql.NullString{String: params.ProductionRunID, Valid: true},
		AccountID:       params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Step 2: BFS traversal following batch flow graph.
	// Matches Dashboard behavior: always traverse downstream (out), only traverse upstream (in) for active branches (open or leading to open batches).
	visited := make(map[string]bool)
	batchIDs := make(map[string]bool)
	activeBranchBatches := make(map[string]bool)

	type queueEntry struct {
		id       string
		isClosed bool
	}

	queue := make([]queueEntry, 0, len(initialRows))
	for _, row := range initialRows {
		queue = append(queue, queueEntry{id: row.ID, isClosed: row.ClosedAt.Valid})
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.id] {
			continue
		}
		visited[current.id] = true
		batchIDs[current.id] = true

		isOpen := !current.isClosed

		// Get outgoing (downstream) batches.
		outgoing, err := r.queries.GetBatchFlowOutgoing(ctx, current.id)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		hasActiveDownstream := false
		for _, outID := range outgoing {
			if activeBranchBatches[outID] {
				hasActiveDownstream = true
				break
			}
		}

		if isOpen || hasActiveDownstream {
			activeBranchBatches[current.id] = true

			// Traverse upstream (in) only for active branches.
			incoming, err := r.queries.GetBatchFlowIncoming(ctx, current.id)
			if apiErr := db.MapSQLError(err); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			for _, inID := range incoming {
				if !visited[inID] {
					closedAt, err := r.queries.GetBatchClosedAt(ctx, sqlc.GetBatchClosedAtParams{
						ID:        inID,
						AccountID: params.AccountID,
					})
					if err != nil {
						continue
					}
					queue = append(queue, queueEntry{id: inID, isClosed: closedAt.Valid})
				}
			}
		}

		// Always traverse downstream (out).
		for _, outID := range outgoing {
			if !visited[outID] {
				closedAt, err := r.queries.GetBatchClosedAt(ctx, sqlc.GetBatchClosedAtParams{
					ID:        outID,
					AccountID: params.AccountID,
				})
				if err != nil {
					continue
				}
				queue = append(queue, queueEntry{id: outID, isClosed: closedAt.Valid})
			}
		}
	}

	// Step 3: Fetch full batch data + machines + lots + flow IDs for all collected IDs.
	batches := make([]*domain.Batch, 0, len(batchIDs))
	for id := range batchIDs {
		row, err := r.queries.GetBatch(ctx, sqlc.GetBatchParams{
			ID:        id,
			AccountID: params.AccountID,
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		batch := mapBatchRow(row)

		machineRows, err := r.queries.GetBatchMachines(ctx, id)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		machines := make([]domain.LightMachine, len(machineRows))
		for i, m := range machineRows {
			machines[i] = domain.LightMachine{ID: m.ID, Name: m.Name}
		}
		batch.Machines = machines

		// Fetch lots (material lots from inventory issues/allocations).
		lotRows, err := r.queries.GetBatchLots(ctx, sqlc.GetBatchLotsParams{
			BatchID: gosql.NullString{String: id, Valid: true},
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		seenLots := make(map[string]bool)
		lots := make([]domain.BatchLot, 0, len(lotRows))
		for _, l := range lotRows {
			if !seenLots[l.LotNumber] {
				seenLots[l.LotNumber] = true
				lots = append(lots, domain.BatchLot{LotNumber: l.LotNumber, Type: l.LotType})
			}
		}
		// Add production run lot number if present.
		if batch.ProductionRun != nil && batch.ProductionRun.Number != "" {
			if !seenLots[batch.ProductionRun.Number] {
				lots = append(lots, domain.BatchLot{LotNumber: batch.ProductionRun.Number, Type: "productionRun"})
			}
		}
		batch.Lots = lots

		// Fetch input/output batch IDs from flow graph.
		incomingIDs, err := r.queries.GetBatchFlowIncoming(ctx, id)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		batch.InputBatchIDs = incomingIDs

		outgoingIDs, err := r.queries.GetBatchFlowOutgoing(ctx, id)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		batch.OutputBatchIDs = outgoingIDs

		batches = append(batches, batch)
	}

	sorted, pageInfo, apiErr := paginateBatchesForProductionRun(batches, params)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return &domain.ListBatchesByProductionRunResult{Batches: sorted, PageInfo: pageInfo}, nil
}

func paginateBatchesForProductionRun(batches []*domain.Batch, params domain.ListBatchesByProductionRunParams) ([]*domain.Batch, pagination.PageInfo, *apierror.APIError) {
	sort.Slice(batches, func(i, j int) bool {
		if !batches[i].CreatedAt.Equal(batches[j].CreatedAt) {
			return batches[i].CreatedAt.After(batches[j].CreatedAt)
		}
		return batches[i].ID > batches[j].ID
	})

	filtered := batches
	if params.SearchQuery != nil && *params.SearchQuery != "" {
		q := strings.ToLower(strings.TrimSpace(*params.SearchQuery))
		out := make([]*domain.Batch, 0, len(batches))
		for _, b := range batches {
			if batchMatchesSearch(b, q) {
				out = append(out, b)
			}
		}
		filtered = out
	}

	start := 0
	var cursorDir *pagination.Direction
	if params.Cursor != nil && *params.Cursor != "" {
		cur, err := pagination.DecodeStringCursor(*params.Cursor)
		if err != nil {
			return nil, pagination.PageInfo{}, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction
		found := false
		for i, b := range filtered {
			if b.ID == cur.ID {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			return []*domain.Batch{}, pagination.PageInfo{}, nil
		}
	}

	lim := params.Limit
	if lim <= 0 {
		lim = 100
	}
	if lim > 1000 {
		lim = 1000
	}

	if start > len(filtered) {
		return []*domain.Batch{}, pagination.PageInfo{}, nil
	}

	window := filtered[start:]
	if len(window) > int(lim)+1 {
		window = window[:lim+1]
	}

	result, pageInfo := pagination.BuildPageString(window, lim, cursorDir,
		func(b *domain.Batch) time.Time { return b.CreatedAt },
		func(b *domain.Batch) string { return b.ID })
	return result, pageInfo, nil
}

func batchMatchesSearch(b *domain.Batch, q string) bool {
	if q == "" {
		return true
	}
	var sb strings.Builder
	sb.WriteString(strings.ToLower(b.ID))
	sb.WriteByte(' ')
	sb.WriteString(strings.ToLower(b.Item.SKU))
	if b.ScanningStation != nil {
		sb.WriteByte(' ')
		sb.WriteString(strings.ToLower(b.ScanningStation.Name))
	}
	if b.DepartmentName != nil {
		sb.WriteByte(' ')
		sb.WriteString(strings.ToLower(*b.DepartmentName))
	}
	if b.ProductionStep != nil {
		sb.WriteByte(' ')
		sb.WriteString(strings.ToLower(b.ProductionStep.Name))
	}
	if b.ProductionRun != nil {
		sb.WriteByte(' ')
		sb.WriteString(strings.ToLower(b.ProductionRun.Number))
		sb.WriteByte(' ')
		sb.WriteString(strings.ToLower(b.ProductionRun.ID))
	}
	for _, lot := range b.Lots {
		sb.WriteByte(' ')
		sb.WriteString(strings.ToLower(lot.LotNumber))
	}
	for _, m := range b.Machines {
		sb.WriteByte(' ')
		sb.WriteString(strings.ToLower(m.Name))
	}
	return strings.Contains(sb.String(), q)
}
