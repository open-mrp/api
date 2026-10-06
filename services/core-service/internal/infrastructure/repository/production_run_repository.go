package repository

import (
	"context"
	gosql "database/sql"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/safeconv"
	"github.com/open-mrp/api/shared/tracing"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/trace"
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
		gosql.NullString{String: db.EscapeLike(*query) + "%", Valid: true}
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

// resolvedResponsibleUserID prefers the account_user id resolved by the query; legacy rows store a user id in responsible_user_id and may have no account_user match, in which case the raw value is kept.
func resolvedResponsibleUserID(accountUserID gosql.NullString, raw string) string {
	if accountUserID.Valid {
		return accountUserID.String
	}
	return raw
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

	summaries, apiErr := r.batchSummariesByRun(ctx, params.AccountID, []string{run.ID})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	run.BatchSummaries = summaries[run.ID]

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

	// Atomic rather than MAX(number)+1: two runs created at once — which releasing two weeks back to back
	// does — read the same maximum and collide on the unique number.
	sysPropertyID, apiErr := id.GenID(id.SysPropertyIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	counter := productionRunNumbers(r.queries, accountID)
	numbers := make([]string, 0, count)
	for range count {
		number, apiErr := counter.next(ctx, sysPropertyID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		numbers = append(numbers, strconv.FormatInt(number, 10))
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

func (r *productionRunRepoImpl) HasScannedBatches(ctx context.Context, accountID, productionRunID string) (bool, *apierror.APIError) {
	ctx, span := productionRunRepoTracer.Start(ctx, "repository.production_run.has_scanned_batches")
	defer span.End()

	hasScanned, err := r.queries.RunHasScannedBatches(ctx, sqlc.RunHasScannedBatchesParams{
		AccountID:       accountID,
		ProductionRunID: gosql.NullString{String: productionRunID, Valid: true},
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, tracing.Trace(span, apiErr)
	}
	return hasScanned, nil
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

	initialRows, err := r.queries.ListRunBatchTraversal(ctx, sqlc.ListRunBatchTraversalParams{
		ProductionRunID: gosql.NullString{String: params.ProductionRunID, Valid: true},
		AccountID:       params.AccountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	initial := make([]batchTraversalRef, len(initialRows))
	for i, row := range initialRows {
		initial[i] = batchTraversalRef{id: row.ID, closed: row.ClosedAt.Valid, createdAt: row.CreatedAt}
	}

	refs := initial
	if params.Scope == nil || *params.Scope != string(constants.ProductionRunBatchScopeRun) {
		walked, apiErr := r.walkBatchFlow(ctx, params.AccountID, initial)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		refs = walked
	}

	// A search matches on hydrated fields (SKU, station, lots, ...), so every candidate is loaded.
	if params.SearchQuery != nil && strings.TrimSpace(*params.SearchQuery) != "" {
		ids := make([]string, len(refs))
		for i, ref := range refs {
			ids[i] = ref.id
		}
		batches, apiErr := r.hydrateBatches(ctx, params.AccountID, ids)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		q := strings.ToLower(strings.TrimSpace(*params.SearchQuery))
		matched := make([]*domain.Batch, 0, len(batches))
		for _, b := range batches {
			if batchMatchesSearch(b, q) {
				matched = append(matched, b)
			}
		}
		page, pageInfo, apiErr := paginateByCreatedAt(matched,
			func(b *domain.Batch) time.Time { return b.CreatedAt },
			func(b *domain.Batch) string { return b.ID },
			params.Cursor, params.Limit)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		return &domain.ListBatchesByProductionRunResult{Batches: page, PageInfo: pageInfo}, nil
	}

	// Otherwise page on the lightweight refs and load only the page.
	pageRefs, pageInfo, apiErr := paginateByCreatedAt(refs,
		func(ref batchTraversalRef) time.Time { return ref.createdAt },
		func(ref batchTraversalRef) string { return ref.id },
		params.Cursor, params.Limit)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	ids := make([]string, len(pageRefs))
	for i, ref := range pageRefs {
		ids[i] = ref.id
	}
	hydrated, apiErr := r.hydrateBatches(ctx, params.AccountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	byID := make(map[string]*domain.Batch, len(hydrated))
	for _, b := range hydrated {
		byID[b.ID] = b
	}
	page := make([]*domain.Batch, 0, len(pageRefs))
	for _, ref := range pageRefs {
		if b, ok := byID[ref.id]; ok {
			page = append(page, b)
		}
	}
	return &domain.ListBatchesByProductionRunResult{Batches: page, PageInfo: pageInfo}, nil
}

// batchTraversalRef is what the flow walk and pagination need of a batch.
type batchTraversalRef struct {
	id        string
	closed    bool
	createdAt time.Time
}

// walkBatchFlow follows the batch flow out from a run's batches: always downstream, and
// upstream only along branches that are open or lead to an open batch. It visits batches in
// the same order as a node-at-a-time breadth-first walk, but loads the edges and closed
// state of every batch waiting in the queue at once, so its queries grow with the depth of
// the flow rather than its size. Batches outside the account are never visited.
func (r *productionRunRepoImpl) walkBatchFlow(ctx context.Context, accountID string, initial []batchTraversalRef) ([]batchTraversalRef, *apierror.APIError) {
	known := make(map[string]batchTraversalRef, len(initial))
	queue := make([]string, 0, len(initial))
	for _, ref := range initial {
		known[ref.id] = ref
		queue = append(queue, ref.id)
	}

	incoming := make(map[string][]string)
	outgoing := make(map[string][]string)
	loaded := make(map[string]bool)
	visited := make(map[string]bool)
	active := make(map[string]bool)
	var result []batchTraversalRef

	for head := 0; head < len(queue); head++ {
		current := queue[head]
		if visited[current] {
			continue
		}

		if !loaded[current] {
			if apiErr := r.loadFlowFrontier(ctx, accountID, queue[head:], visited, loaded, known, incoming, outgoing); apiErr != nil {
				return nil, apiErr
			}
		}

		visited[current] = true
		result = append(result, known[current])

		hasActiveDownstream := false
		for _, out := range outgoing[current] {
			if active[out] {
				hasActiveDownstream = true
				break
			}
		}
		if !known[current].closed || hasActiveDownstream {
			active[current] = true
			for _, in := range incoming[current] {
				if _, inAccount := known[in]; inAccount && !visited[in] {
					queue = append(queue, in)
				}
			}
		}
		for _, out := range outgoing[current] {
			if _, inAccount := known[out]; inAccount && !visited[out] {
				queue = append(queue, out)
			}
		}
	}
	return result, nil
}

// loadFlowFrontier loads, in two queries, the edges of every queued batch not yet loaded and
// the closed state of each neighbour those edges reach.
func (r *productionRunRepoImpl) loadFlowFrontier(
	ctx context.Context, accountID string, pending []string,
	visited, loaded map[string]bool, known map[string]batchTraversalRef,
	incoming, outgoing map[string][]string,
) *apierror.APIError {
	frontier := make([]string, 0, len(pending))
	for _, id := range pending {
		if !visited[id] && !loaded[id] {
			loaded[id] = true
			frontier = append(frontier, id)
		}
	}

	edges, err := r.queries.ListBatchFlowEdgesForBatches(ctx, sqlc.ListBatchFlowEdgesForBatchesParams{DownstreamIds: frontier, UpstreamIds: frontier})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	inFrontier := make(map[string]bool, len(frontier))
	for _, id := range frontier {
		inFrontier[id] = true
	}
	var unknown []string
	seen := make(map[string]bool)
	note := func(id string) {
		if _, ok := known[id]; !ok && !seen[id] {
			seen[id] = true
			unknown = append(unknown, id)
		}
	}
	for _, e := range edges {
		if inFrontier[e.DownstreamID] {
			incoming[e.DownstreamID] = append(incoming[e.DownstreamID], e.UpstreamID)
			note(e.UpstreamID)
		}
		if inFrontier[e.UpstreamID] {
			outgoing[e.UpstreamID] = append(outgoing[e.UpstreamID], e.DownstreamID)
			note(e.DownstreamID)
		}
	}

	if len(unknown) == 0 {
		return nil
	}
	rows, err := r.queries.ListBatchTraversalByIDs(ctx, sqlc.ListBatchTraversalByIDsParams{Ids: unknown, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	for _, row := range rows {
		known[row.ID] = batchTraversalRef{id: row.ID, closed: row.ClosedAt.Valid, createdAt: row.CreatedAt}
	}
	return nil
}

// paginateByCreatedAt orders items newest first (ties by ID) and cuts the page after the cursor.
func paginateByCreatedAt[T any](items []T, createdAt func(T) time.Time, idOf func(T) string, cursor *string, limit int32) ([]T, pagination.PageInfo, *apierror.APIError) {
	sort.Slice(items, func(i, j int) bool {
		if !createdAt(items[i]).Equal(createdAt(items[j])) {
			return createdAt(items[i]).After(createdAt(items[j]))
		}
		return idOf(items[i]) > idOf(items[j])
	})

	start := 0
	var cursorDir *pagination.Direction
	if cursor != nil && *cursor != "" {
		cur, err := pagination.DecodeStringCursor(*cursor)
		if err != nil {
			return nil, pagination.PageInfo{}, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cursorDir = &cur.Direction
		found := false
		for i, item := range items {
			if idOf(item) == cur.ID {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			return []T{}, pagination.PageInfo{}, nil
		}
	}

	lim := limit
	if lim <= 0 {
		lim = 100
	}
	if lim > 1000 {
		lim = 1000
	}

	window := items[start:]
	if len(window) > int(lim)+1 {
		window = window[:lim+1]
	}

	page, pageInfo := pagination.BuildPageString(window, lim, cursorDir, createdAt, idOf)
	return page, pageInfo, nil
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

// hydrateBatches loads full batches with their machines, lots, and flow neighbours in a
// fixed number of queries, however many batches are asked for.
func (r *productionRunRepoImpl) hydrateBatches(ctx context.Context, accountID string, ids []string) ([]*domain.Batch, *apierror.APIError) {
	if len(ids) == 0 {
		return []*domain.Batch{}, nil
	}

	rows, err := r.queries.ListBatchesByIDs(ctx, sqlc.ListBatchesByIDsParams{Ids: ids, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	// Only batches that proved to be in the account get their relations loaded.
	found := make([]string, len(rows))
	nullIDs := make([]gosql.NullString, len(rows))
	for i, row := range rows {
		found[i] = row.ID
		nullIDs[i] = gosql.NullString{String: row.ID, Valid: true}
	}
	if len(found) == 0 {
		return []*domain.Batch{}, nil
	}

	edges, err := r.queries.ListBatchFlowEdgesForBatches(ctx, sqlc.ListBatchFlowEdgesForBatchesParams{DownstreamIds: found, UpstreamIds: found})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	inputsByBatch := make(map[string][]string)
	outputsByBatch := make(map[string][]string)
	for _, e := range edges {
		inputsByBatch[e.DownstreamID] = append(inputsByBatch[e.DownstreamID], e.UpstreamID)
		outputsByBatch[e.UpstreamID] = append(outputsByBatch[e.UpstreamID], e.DownstreamID)
	}

	batches := make([]*domain.Batch, len(rows))
	for i, row := range rows {
		batch := mapBatchRow(sqlc.GetBatchRow(row))
		batch.InputBatchIDs = inputsByBatch[row.ID]
		batch.OutputBatchIDs = outputsByBatch[row.ID]
		batches[i] = batch
	}
	if apiErr := attachMachinesAndLots(ctx, r.queries, batches); apiErr != nil {
		return nil, apiErr
	}
	return batches, nil
}

// batchSummariesByRun totals the given runs' batches per item and unit, in one query.
func (r *productionRunRepoImpl) batchSummariesByRun(ctx context.Context, accountID string, runIDs []string) (map[string][]domain.ProductionRunBatchSummary, *apierror.APIError) {
	out := make(map[string][]domain.ProductionRunBatchSummary, len(runIDs))
	if len(runIDs) == 0 {
		return out, nil
	}
	ids := make([]gosql.NullString, len(runIDs))
	for i, id := range runIDs {
		ids[i] = gosql.NullString{String: id, Valid: true}
		out[id] = []domain.ProductionRunBatchSummary{}
	}

	rows, err := r.queries.ListProductionRunBatchSummaries(ctx, sqlc.ListProductionRunBatchSummariesParams{
		AccountID:        accountID,
		ProductionRunIds: ids,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, apiErr
	}
	for _, row := range rows {
		// sqlc cannot type a CAST, so the driver's []byte comes through as any.
		raw, _ := row.QuantityValue.([]byte)
		quantity, err := decimal.NewFromString(string(raw))
		if err != nil {
			return nil, apierror.NewInternalError(err, "Issue reading a production run's batch total.")
		}
		runID := row.ProductionRunID.String
		out[runID] = append(out[runID], domain.ProductionRunBatchSummary{
			ItemID:           row.ItemID,
			ItemSKU:          row.ItemSku,
			UnitID:           row.UnitID,
			UnitAbbreviation: row.UnitAbbreviation,
			Quantity:         quantity,
			BatchCount:       safeconv.Int64ToInt32(row.BatchCount),
		})
	}
	return out, nil
}

// listResultWithSummaries attaches each listed run's batch totals.
func (r *productionRunRepoImpl) listResultWithSummaries(ctx context.Context, span trace.Span, accountID string, runs []*domain.ProductionRunSummary, pageInfo pagination.PageInfo) (*domain.ListProductionRunsResult, *apierror.APIError) {
	ids := make([]string, len(runs))
	for i, run := range runs {
		ids[i] = run.ID
	}
	summaries, apiErr := r.batchSummariesByRun(ctx, accountID, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, run := range runs {
		run.BatchSummaries = summaries[run.ID]
	}
	return &domain.ListProductionRunsResult{ProductionRuns: runs, PageInfo: pageInfo}, nil
}
