package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/excel"
	"github.com/open-mrp/api/shared/field"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/tracing"
)

// asyncBulkDeps hands the async bulk engine the plumbing it runs on.
func (s *locationSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

// upsertLocationInTx creates or updates a location's name and type in one transaction. An
// update preserves the existing parent_id; parent/child links are applied by the caller
// after the upsert. It returns the upserted id.
func upsertLocationInTx(txCtx context.Context, txRepos domain.RepoFactory, accountID string, row domain.ResolvedUpsertLocationRow, oldLocation *domain.Location) (*string, *apierror.APIError) {
	ctx, span := locationSvcTracer.Start(txCtx, "service.location.upsert_in_tx")
	defer span.End()

	txRepo := txRepos.NewLocationRepo()

	var upsertID string

	if oldLocation == nil { // create — links applied by the caller
		locationID, apiErr := id.GenID(id.LocationIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		created, apiErr := txRepo.Create(ctx, locationID, domain.CreateLocationParams{
			AccountID: accountID,
			Name:      row.Name,
			TypeCode:  row.TypeCode,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		upsertID = created.ID

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeLocation,
			ResourceID:   created.ID,
			Changes:      changes,
		}); apiErr != nil {
			return nil, apiErr
		}

	} else { // update — name + type_code only; preserve existing parent_id
		// Preserve the existing parent_id here; the caller re-applies incoming links after.
		parentID := field.Unset[string]()
		if oldLocation.ParentID != nil {
			parentID = field.Set(*oldLocation.ParentID)
		}

		updated, apiErr := txRepo.Update(ctx, domain.UpdateLocationParams{
			AccountID:  accountID,
			LocationID: oldLocation.ID,
			Name:       &row.Name,
			TypeCode:   &row.TypeCode,
			ParentID:   parentID,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		upsertID = updated.ID

		changes := audit.ComputeChanges(oldLocation, updated)
		if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeLocation,
			ResourceID:   updated.ID,
			Changes:      changes,
		}); apiErr != nil {
			return nil, apiErr
		}
	}

	return &upsertID, nil
}

// validateBulkUpsertLocationRows runs the accept-phase structural checks: no duplicate
// location name within the request (case-insensitive). It touches no database.
func validateBulkUpsertLocationRows(rows []domain.UpsertLocationParams) *apierror.APIError {
	nameInputSpace := make(map[string]struct{}, len(rows))
	var rowErrs apierror.RowErrors
	for i, loc := range rows {
		lower := strings.ToLower(loc.Name)
		if _, exists := nameInputSpace[lower]; exists {
			rowErrs.AddValidation(i, fmt.Sprintf("locations[%d].name", i), fmt.Sprintf("duplicate name %q in request", loc.Name))
		}
		nameInputSpace[lower] = struct{}{}
	}
	return rowErrs.Summary("locations")
}

// resolveBulkUpsertLocationRows resolves every row's fuzzy parent and child references. A
// reference whose name matches another row in the same batch is an intra-batch link, kept
// as a batch-name reference for the write phase to resolve once every row has an id — this
// is what lets an import define a whole hierarchy (parents/children pointing at siblings)
// in one request. Every other reference must resolve against an existing location, failing
// fast with a row-indexed 400 if it does not.
func resolveBulkUpsertLocationRows(ctx context.Context, repos domain.RepoFactory, accountID string, rows []domain.UpsertLocationParams) ([]domain.ResolvedUpsertLocationRow, *apierror.APIError) {
	repo := repos.NewLocationRepo()

	// Names of the rows in this batch (lowercased). A reference by name to one of these is
	// an intra-batch link — resolved to the row's id in the write phase, not here.
	batchNames := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		batchNames[strings.ToLower(r.Name)] = struct{}{}
	}
	isIntraBatch := func(ref domain.ObjectIdentifier) bool {
		if ref.ID != "" || ref.Name == "" {
			return false
		}
		_, ok := batchNames[strings.ToLower(ref.Name)]
		return ok
	}

	// Every reference that is not intra-batch must resolve against an existing location.
	// Collect them all and build one resolver.
	var dbRefs []domain.ObjectIdentifier
	for i := range rows {
		if rows[i].Parent != nil && !isIntraBatch(*rows[i].Parent) {
			dbRefs = append(dbRefs, *rows[i].Parent)
		}
		for _, c := range rows[i].Children {
			if !isIntraBatch(c) {
				dbRefs = append(dbRefs, c)
			}
		}
	}
	resolver, apiErr := newObjectIdentifierResolver(
		ctx, accountID, "location", dbRefs, repo.GetByIDs, repo.FindByNames,
		func(l *domain.Location) string { return l.ID },
		func(l *domain.Location) string { return l.Name },
	)
	if apiErr != nil {
		return nil, apiErr
	}

	// toRef classifies one reference: an intra-batch name becomes a batch-name reference,
	// anything else must resolve to an existing location id (row-indexed 400 otherwise).
	toRef := func(ref domain.ObjectIdentifier, param string) (domain.LocationRef, *apierror.APIError) {
		if isIntraBatch(ref) {
			return domain.LocationRef{BatchName: strings.ToLower(ref.Name)}, nil
		}
		id, apiErr := resolver.resolveOrError(ref, param)
		if apiErr != nil {
			return domain.LocationRef{}, apiErr
		}
		return domain.LocationRef{ExistingID: id}, nil
	}

	resolved := make([]domain.ResolvedUpsertLocationRow, len(rows))
	for i := range rows {
		loc := rows[i]

		var parent *domain.LocationRef
		if loc.Parent != nil {
			ref, apiErr := toRef(*loc.Parent, fmt.Sprintf("locations[%d].parent", i))
			if apiErr != nil {
				return nil, apiErr
			}
			parent = &ref
		}

		children := make([]domain.LocationRef, 0, len(loc.Children))
		for j, c := range loc.Children {
			ref, apiErr := toRef(c, fmt.Sprintf("locations[%d].children[%d]", i, j))
			if apiErr != nil {
				return nil, apiErr
			}
			children = append(children, ref)
		}

		resolved[i] = domain.ResolvedUpsertLocationRow{
			Name:     loc.Name,
			TypeCode: loc.TypeCode,
			Parent:   parent,
			Children: children,
		}
	}
	return resolved, nil
}

// writeBulkUpsertLocations is the engine's Write hook, in two phases because a row's links
// may point at sibling rows that do not exist yet:
//
//   - Phase 1 upserts every row's name and type inside its own savepoint (partial success),
//     recording each success under its name so intra-batch links can be resolved.
//   - Phase 2 applies each surviving row's parent and child links in its own savepoint,
//     resolving a batch-name reference to the id phase 1 assigned it.
//
// A phase-1 failure drops the row (recorded in errors, excluded from the name→id map, so
// links to it can't resolve). A phase-2 link failure is recorded in errors too; the row's
// upsert stands, so a created-but-unlinked location appears in both results and errors.
func writeBulkUpsertLocations(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.ResolvedUpsertLocationRow) (BulkWriteResult, *apierror.APIError) {
	names := make([]string, len(rows))
	for i, r := range rows {
		names[i] = strings.ToLower(r.Name)
	}

	txRepo := txRepos.NewLocationRepo()
	existingLocations, apiErr := txRepo.FindByNames(txCtx, accountID, names)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	locationsByName := make(map[string]*domain.Location, len(existingLocations))
	for _, loc := range existingLocations {
		locationsByName[strings.ToLower(loc.Name)] = loc
	}

	// ── Phase 1: upsert name + type; record each row's id by name ───────────────────────
	idByIndex := make([]string, len(rows))
	isCreate := make([]bool, len(rows))
	upserted := make([]bool, len(rows))
	idByName := make(map[string]string, len(rows))
	var rowErrors []apierror.RowError

	for i := range rows {
		row := rows[i]
		var upID string
		var create bool
		rowErr := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
			old := locationsByName[names[i]]
			id, apiErr := upsertLocationInTx(spCtx, txRepos, accountID, row, old)
			if apiErr != nil {
				return apiErr
			}
			if id == nil {
				return apierror.NewInvariantViolationError(fmt.Sprintf("problem upserting location, no ID returned: %+v", row))
			}
			upID = *id
			create = old == nil
			return nil
		})
		if rowErr != nil {
			rowErrors = append(rowErrors, apierror.NewRowError(i, rowErr))
			continue
		}
		upserted[i] = true
		idByIndex[i] = upID
		isCreate[i] = create
		idByName[names[i]] = upID
	}

	// resolveRef turns a resolved reference into a concrete location id: an existing id
	// passes through; a batch-name reference resolves to the id phase 1 assigned that row,
	// or reports missing if that row failed phase 1.
	resolveRef := func(ref domain.LocationRef) (string, bool) {
		if ref.ExistingID != "" {
			return ref.ExistingID, true
		}
		id, ok := idByName[ref.BatchName]
		return id, ok
	}

	// ── Phase 2: apply parent/child links now that every row has an id ───────────────────
	for i := range rows {
		if !upserted[i] {
			continue
		}
		row := rows[i]
		thisID := idByIndex[i]
		rowErr := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
			if row.Parent != nil {
				parentID, ok := resolveRef(*row.Parent)
				if !ok {
					return apierror.NewValidationError("Parent location was not created in this batch.")
				}
				if apiErr := txRepo.LinkParent(spCtx, accountID, thisID, parentID); apiErr != nil {
					return apiErr
				}
			}
			for _, c := range row.Children {
				childID, ok := resolveRef(c)
				if !ok {
					return apierror.NewValidationError("Child location was not created in this batch.")
				}
				if apiErr := txRepo.LinkParent(spCtx, accountID, childID, thisID); apiErr != nil {
					return apiErr
				}
			}
			return nil
		})
		if rowErr != nil {
			rowErrors = append(rowErrors, apierror.NewRowError(i, rowErr))
		}
	}

	results := make([]domain.RowResult, 0, len(rows))
	for i := range rows {
		if !upserted[i] {
			continue
		}
		results = append(results, newRowResult(i, idByIndex[i], isCreate[i]))
	}

	return BulkWriteResult{Results: results, Errors: rowErrors, WrittenIDs: resultIDs(results)}, nil
}

// bulkUpsertSpec wires locations into the async bulk engine.
func (s *locationSvcImpl) bulkUpsertSpec() bulkOperationSpec[domain.UpsertLocationParams, domain.ResolvedUpsertLocationRow] {
	return bulkOperationSpec[domain.UpsertLocationParams, domain.ResolvedUpsertLocationRow]{
		JobType:          constants.JobTypeBulkUpsert,
		ResourceType:     constants.ObjectTypeLocation,
		RoutingKey:       messaging.BulkUpsertLocations.RoutingKey(),
		PermissionDomain: types.PermissionDomainLocations,
		Actions:          []types.Action{types.ActionCreate, types.ActionUpdate},
		EntityName:       "locations",
		Validate:         validateBulkUpsertLocationRows,
		Resolve:          resolveBulkUpsertLocationRows,
		Write:            writeBulkUpsertLocations,
	}
}

// BulkUpsertLocations accepts a bulk upsert: it validates and resolves synchronously,
// records the resolved rows on a job, and returns the raised job to poll. The locations are
// created or updated asynchronously by ExecuteBulkUpsertLocations.
func (s *locationSvcImpl) BulkUpsertLocations(ctx context.Context, params domain.BulkUpsertLocationsParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), params.Locations)
}

// ExecuteBulkUpsertLocations performs the writes for an enqueued bulk upsert. Called by the
// bulk upsert consumer; exactly-once is provided by the message inbox.
func (s *locationSvcImpl) ExecuteBulkUpsertLocations(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), event)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// hands the export engine the plumbing it runs on.

// wires storage locations into the export engine.
func (s *locationSvcImpl) exportSpec() exportSpec[*domain.Location, domain.ExportLocationsParams] {
	return exportSpec[*domain.Location, domain.ExportLocationsParams]{
		PermissionDomain: types.PermissionDomainLocations,
		Name:             "Storage Locations",
		Slug:             "locations",
		ResourceType:     constants.ObjectTypeLocation,
		Columns: []excel.ColumnSpec{
			{Header: "ID", Key: "id", Width: 6},
			{Header: "Name", Key: "name", Width: 25},
			{Header: "Type", Key: "type", Width: 14},
			{Header: "Parent", Key: "parent", Width: 25},
			{Header: "Children", Key: "children", Width: 50},
		},

		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportLocationsParams) ([]*domain.Location, *apierror.APIError) {
			filters.AccountID = accountID
			return repos.NewLocationRepo().Export(ctx, filters)
		},

		Project: func(location *domain.Location) excel.Row {
			children := make([]string, len(location.Children))
			for i, c := range location.Children {
				children[i] = c.Name
			}
			return excel.Row{
				"id":       location.ID,
				"name":     location.Name,
				"type":     location.TypeCode,
				"parent":   excel.Str(location.ParentName),
				"children": excel.JoinNames(children),
			}
		},
	}
}

// accepts an export: records what to build on a job and returns it to poll.
func (s *locationSvcImpl) ExportLocations(ctx context.Context, params domain.ExportLocationsParams) (*domain.Job, *apierror.APIError) {
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), params)
}

// renders the file an accepted export recorded
func (s *locationSvcImpl) BuildExportLocations(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}
