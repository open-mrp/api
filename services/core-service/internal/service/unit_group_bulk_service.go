package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/excel"
	"github.com/augno/api/shared/field"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

// asyncBulkDeps hands the async bulk engine the plumbing it runs on.
func (s *unitGroupSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

func upsertUnitGroupInTx(txCtx context.Context, txRepos domain.RepoFactory, accountID string, row domain.ResolvedUpsertUnitGroupRow, oldGroup *domain.UnitGroupFull) (*string, *apierror.APIError) {
	ctx, span := unitGroupSvcTracer.Start(txCtx, "service.unit_group.upsert_in_tx")
	defer span.End()
	txRepo := txRepos.NewUnitGroupRepo()

	// The caller resolved every unit reference and checked each conversion's dimension
	// against the group's type, so no per-unit lookup is needed here.

	var upsertID string
	if oldGroup == nil { // create
		// Guard against a concurrent create slipping in between the pre-flight FindByNames
		// check (outside the transaction) and this insert. There is no unique DB constraint
		// on (account_id, name), so the application check is the only guard.
		exists, apiErr := txRepo.ExistsByName(ctx, accountID, row.Name, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		if exists {
			return nil, tracing.Trace(span, apierror.NewConflictErrorWithParam(
				fmt.Sprintf("A unit group with name %q already exists.", row.Name), "name"))
		}

		ugID, apiErr := id.GenID(id.UnitGroupIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		if _, apiErr := txRepo.Create(ctx, ugID, domain.CreateUnitGroupParams{
			AccountID:  accountID,
			Name:       row.Name,
			Notes:      row.Notes,
			Type:       row.Type,
			BaseUnitID: row.BaseUnitID,
		}); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		writtenUnitIDs := make(map[string]struct{}, len(row.Conversions))
		for _, conv := range row.Conversions {
			uguID, apiErr := id.GenID(id.UnitGroupsUnitsIDPrefix, nil)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			if _, apiErr := txRepo.UpsertUnitGroupUnit(ctx, uguID, domain.UpsertUnitGroupUnitParams{
				AccountID:          accountID,
				UnitGroupID:        ugID,
				UnitGroupUnitID:    uguID,
				UnitID:             conv.UnitID,
				DiscountPercentage: conv.DiscountPercentage,
				DiscountFixed:      "0",
				IsVisible:          true,
			}); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			writtenUnitIDs[conv.UnitID] = struct{}{}
		}

		// The base unit is always an associated unit, even when not listed.
		if apiErr := ensureBaseUnitInGroupInTx(ctx, txRepos, accountID, ugID, row.BaseUnitID, writtenUnitIDs); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		created, apiErr := txRepo.Get(ctx, domain.GetUnitGroupParams{
			AccountID:   accountID,
			UnitGroupID: ugID,
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		upsertID = created.ID

		changes := audit.ComputeChanges(nil, created)
		if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeUnitGroup,
			ResourceID:   created.ID,
			Changes:      changes,
		}); apiErr != nil {
			return nil, apiErr
		}

		return &upsertID, nil
	}

	// update
	if oldGroup.AccountID == nil {
		return nil, tracing.Trace(span, apierror.NewValidationErrorWithParam(
			fmt.Sprintf("System unit group %q cannot be modified.", row.Name), "name"))
	}

	existingByUnitID := make(map[string]*domain.UnitGroupUnit, len(oldGroup.UnitConversions))
	for _, u := range oldGroup.UnitConversions {
		existingByUnitID[u.UnitID] = u
	}

	// Build the set of incoming unit IDs so we can diff against the existing set.
	incomingUnitIDs := make(map[string]struct{}, len(row.Conversions))
	for _, conv := range row.Conversions {
		incomingUnitIDs[conv.UnitID] = struct{}{}
	}

	// Delete conversions that are present in the existing group but absent from the
	// incoming list, preserving individual audit records and deleted_record entries.
	// The base unit is never deleted — a group must always include its base unit.
	for _, existingUnit := range oldGroup.UnitConversions {
		if _, keep := incomingUnitIDs[existingUnit.UnitID]; keep {
			continue
		}
		if existingUnit.UnitID == row.BaseUnitID {
			continue
		}

		if apiErr := txRepos.NewDeletedRecordRepo().Create(ctx,
			constants.DeletedRecordResourceTypeUnitGroupUnit, existingUnit.ID, existingUnit,
		); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		if apiErr := txRepo.DeleteUnitGroupUnit(ctx, domain.DeleteUnitGroupUnitParams{
			AccountID:       accountID,
			UnitGroupID:     oldGroup.ID,
			UnitGroupUnitID: existingUnit.ID,
		}); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}

		deletionChanges := audit.ComputeChanges(existingUnit, (*domain.UnitGroupUnit)(nil))
		if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeUnitGroupUnit,
			ResourceID:   existingUnit.ID,
			Changes:      deletionChanges,
		}); apiErr != nil {
			return nil, apiErr
		}
	}

	// an omitted note leaves the stored one alone
	notes := field.Unset[string]()
	if row.Notes != nil {
		notes = field.Set(*row.Notes)
	}
	if _, apiErr := txRepo.Update(ctx, domain.UpdateUnitGroupParams{
		AccountID:   accountID,
		UnitGroupID: oldGroup.ID,
		Name:        &row.Name,
		Notes:       notes,
		BaseUnitID:  &row.BaseUnitID,
	}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Upsert each incoming conversion, preserving the existing row's ID when the
	// unit_id already exists so that IDs remain stable across bulk updates.
	for _, conv := range row.Conversions {
		discountFixed := "0"
		isVisible := true
		var uguID string
		if existing, ok := existingByUnitID[conv.UnitID]; ok {
			discountFixed = existing.DiscountFixed
			isVisible = existing.IsVisible
			uguID = existing.ID
		} else {
			genID, apiErr := id.GenID(id.UnitGroupsUnitsIDPrefix, nil)
			if apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
			uguID = genID
		}

		if _, apiErr := txRepo.UpsertUnitGroupUnit(ctx, uguID, domain.UpsertUnitGroupUnitParams{
			AccountID:          accountID,
			UnitGroupID:        oldGroup.ID,
			UnitGroupUnitID:    uguID,
			UnitID:             conv.UnitID,
			DiscountPercentage: conv.DiscountPercentage,
			DiscountFixed:      discountFixed,
			IsVisible:          isVisible,
		}); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// The base unit is always an associated unit: count both the incoming conversions
	// and any pre-existing association (protected from the deletion diff above).
	writtenUnitIDs := make(map[string]struct{}, len(incomingUnitIDs)+1)
	for uid := range incomingUnitIDs {
		writtenUnitIDs[uid] = struct{}{}
	}
	if _, ok := existingByUnitID[row.BaseUnitID]; ok {
		writtenUnitIDs[row.BaseUnitID] = struct{}{}
	}
	if apiErr := ensureBaseUnitInGroupInTx(ctx, txRepos, accountID, oldGroup.ID, row.BaseUnitID, writtenUnitIDs); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	updated, apiErr := txRepo.Get(ctx, domain.GetUnitGroupParams{
		AccountID:   accountID,
		UnitGroupID: oldGroup.ID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	upsertID = updated.ID

	changes := audit.ComputeChanges(oldGroup, updated)
	if apiErr := audit.NewPublisher().Publish(ctx, txRepos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeUnitGroup,
		ResourceID:   updated.ID,
		Changes:      changes,
	}); apiErr != nil {
		return nil, apiErr
	}

	return &upsertID, nil
}

// validateBulkUpsertUnitGroupRows runs the accept-phase structural checks: no duplicate
// group name within the request (case-insensitive). It touches no database.
func validateBulkUpsertUnitGroupRows(rows []domain.UpsertUnitGroupParams) *apierror.APIError {
	nameInputSpace := make(map[string]struct{}, len(rows))
	var rowErrs apierror.RowErrors
	for i, unitGroup := range rows {
		lowerName := strings.ToLower(unitGroup.Name)
		if _, exists := nameInputSpace[lowerName]; exists {
			rowErrs.AddValidation(i, fmt.Sprintf("unit_groups[%d].name", i), fmt.Sprintf("duplicate name %q in request", unitGroup.Name))
		}
		nameInputSpace[lowerName] = struct{}{}
	}
	return rowErrs.Summary("unit groups")
}

// resolveBulkUpsertUnitGroupRows resolves every group's base unit and conversions to ids
// in one pass, carrying each unit's dimension for the group-type check that Write does.
// Unresolvable references and a unit listed twice in the same group fail fast with a
// row-indexed 400. The dimension check needs the group's stored (immutable) type, so it
// is deferred to Write against existing rows.
func resolveBulkUpsertUnitGroupRows(ctx context.Context, repos domain.RepoFactory, accountID string, rows []domain.UpsertUnitGroupParams) ([]domain.ResolvedUpsertUnitGroupRow, *apierror.APIError) {
	var unitIdentifiers []domain.UnitIdentifier
	for _, ug := range rows {
		unitIdentifiers = append(unitIdentifiers, ug.BaseUnit)
		for _, conv := range ug.UnitConversions {
			unitIdentifiers = append(unitIdentifiers, conv.Unit)
		}
	}
	units, apiErr := newUnitIdentifierResolver(ctx, repos, accountID, unitIdentifiers)
	if apiErr != nil {
		return nil, apiErr
	}

	resolved := make([]domain.ResolvedUpsertUnitGroupRow, len(rows))
	for i := range rows {
		ug := rows[i]

		baseUnit, apiErr := units.resolveOrError(ug.BaseUnit, fmt.Sprintf("unit_groups[%d].base_unit", i))
		if apiErr != nil {
			return nil, apiErr
		}

		conversions := make([]domain.ResolvedUnitGroupConversion, 0, len(ug.UnitConversions))
		seenUnitIDs := make(map[string]struct{}, len(ug.UnitConversions))
		for j, conv := range ug.UnitConversions {
			param := fmt.Sprintf("unit_groups[%d].unit_conversions[%d].unit", i, j)
			unit, apiErr := units.resolveOrError(conv.Unit, param)
			if apiErr != nil {
				return nil, apiErr
			}
			// Duplicates are detected on the resolved id, so naming the same unit once by
			// id and once by name is caught too.
			if _, dup := seenUnitIDs[unit.ID]; dup {
				return nil, apierror.NewValidationErrorWithParam(
					fmt.Sprintf("Duplicate unit in unit group %q.", ug.Name), param)
			}
			seenUnitIDs[unit.ID] = struct{}{}
			conversions = append(conversions, domain.ResolvedUnitGroupConversion{
				UnitID:             unit.ID,
				DimensionCode:      unit.DimensionCode,
				DiscountPercentage: conv.DiscountPercentage,
			})
		}

		resolved[i] = domain.ResolvedUpsertUnitGroupRow{
			Name:                  ug.Name,
			Notes:                 ug.Notes,
			Type:                  ug.Type,
			BaseUnitID:            baseUnit.ID,
			BaseUnitDimensionCode: baseUnit.DimensionCode,
			Conversions:           conversions,
		}
	}
	return resolved, nil
}

// bulkUpsertSpec wires unit groups into the async bulk engine.
func (s *unitGroupSvcImpl) bulkUpsertSpec() bulkOperationSpec[domain.UpsertUnitGroupParams, domain.ResolvedUpsertUnitGroupRow] {
	return bulkOperationSpec[domain.UpsertUnitGroupParams, domain.ResolvedUpsertUnitGroupRow]{
		JobType:          constants.JobTypeBulkUpsert,
		RoutingKey:       messaging.BulkUpsertUnitGroups.RoutingKey(),
		PermissionDomain: types.PermissionDomainUnitGroups,
		Actions:          []types.Action{types.ActionCreate, types.ActionUpdate},
		EntityName:       "unit groups",
		Validate:         validateBulkUpsertUnitGroupRows,
		Resolve:          resolveBulkUpsertUnitGroupRows,
		Write:            writeBulkUpsertUnitGroups,
	}
}

// BulkUpsertUnitGroups accepts a bulk upsert: it validates and resolves synchronously,
// records the resolved rows on a job, and returns the raised job to poll. The groups are
// created or updated asynchronously by ExecuteBulkUpsertUnitGroups.
func (s *unitGroupSvcImpl) BulkUpsertUnitGroups(ctx context.Context, params domain.BulkUpsertUnitGroupsParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), params.UnitGroups)
}

// ExecuteBulkUpsertUnitGroups performs the writes for an enqueued bulk upsert. Called by
// the bulk upsert consumer; exactly-once is provided by the message inbox.
func (s *unitGroupSvcImpl) ExecuteBulkUpsertUnitGroups(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), event)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// hands the export engine the plumbing it runs on.

// wires unit groups into the export engine. A group's associated units are listed
// one per row, so the group's own columns sit on the first of them.
func (s *unitGroupSvcImpl) exportSpec() exportSpec[*domain.UnitGroupFull, domain.ExportUnitGroupsParams] {
	return exportSpec[*domain.UnitGroupFull, domain.ExportUnitGroupsParams]{
		PermissionDomain: types.PermissionDomainUnitGroups,
		Name:             "Unit Groups",
		Slug:             "unit_groups",
		Columns: []excel.ColumnSpec{
			{Header: "ID", Key: "id", Width: 6},
			{Header: "Name", Key: "name", Width: 22},
			{Header: "Type", Key: "type", Width: 14},
			{Header: "Base Unit", Key: "base_unit", Width: 22},
			{Header: "Units", Key: "units", Width: 22},
			{Header: "Discount %'s", Key: "discount_percentage", Width: 14},
			{Header: "Notes", Key: "notes", Width: 35},
		},

		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportUnitGroupsParams) ([]*domain.UnitGroupFull, *apierror.APIError) {
			filters.AccountID = accountID
			return repos.NewUnitGroupRepo().Export(ctx, filters)
		},

		Expand: func(group *domain.UnitGroupFull) []excel.Row {
			parent := excel.Row{
				"id":        group.ID,
				"name":      group.Name,
				"type":      group.Type,
				"base_unit": unitLabel(group.BaseUnit.Name, group.BaseUnit.Abbreviation),
				"notes":     excel.Str(group.Notes),
			}

			children := make([]excel.Row, 0, len(group.UnitConversions))
			for _, uc := range group.UnitConversions {
				// The base unit has its own column; the importer re-adds it to the
				// conversions, so listing it again would duplicate it on re-import.
				if uc.UnitID == group.BaseUnit.ID {
					continue
				}
				children = append(children, excel.Row{
					"units":               unitLabel(uc.Unit.Name, uc.Unit.Abbreviation),
					"discount_percentage": discountPercent(uc.DiscountPercentage),
				})
			}

			return excel.Group(parent, children)
		},
	}
}

// labels a unit the way the importer parses it back
func unitLabel(name, abbreviation string) string {
	return name + ", " + abbreviation
}

// renders a stored discount fraction as the percentage a reader expects, with no
// trailing zeros; a zero discount stays blank so re-import keeps the default
func discountPercent(stored string) string {
	fraction, err := strconv.ParseFloat(stored, 64)
	if err != nil || fraction == 0 {
		return ""
	}
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(fraction*100, 'f', 2, 64), "0"), ".")
}

// accepts an export: records what to build on a job and returns it to poll.
func (s *unitGroupSvcImpl) ExportUnitGroups(ctx context.Context, params domain.ExportUnitGroupsParams) (*domain.Job, *apierror.APIError) {
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), params)
}

// renders the file an accepted export recorded
func (s *unitGroupSvcImpl) BuildExportUnitGroups(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}

// writeBulkUpsertUnitGroups is the engine's Write hook: it bulk-reads existing groups and
// their conversions, then upserts each row inside its own savepoint (partial success). A
// group whose conversion dimension does not match its type, or that fails to write, rolls
// back only itself and is recorded in errors.
func writeBulkUpsertUnitGroups(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.ResolvedUpsertUnitGroupRow) (BulkWriteResult, *apierror.APIError) {
	names := make([]string, len(rows))
	for i, r := range rows {
		names[i] = strings.ToLower(r.Name)
	}

	ugRepo := txRepos.NewUnitGroupRepo()
	existingUnitGroups, apiErr := ugRepo.FindByNames(txCtx, accountID, names)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}

	existingGroupIDs := make([]string, 0, len(existingUnitGroups))
	for _, ug := range existingUnitGroups {
		if ug.ID == "" {
			return BulkWriteResult{}, apierror.NewInvariantViolationError(fmt.Sprintf("Empty ID in unit group: %+v", ug))
		}
		existingGroupIDs = append(existingGroupIDs, ug.ID)
	}

	allUnits, apiErr := ugRepo.FindUnitsByGroupIDs(txCtx, existingGroupIDs)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}
	unitsByGroupID := make(map[string][]*domain.UnitGroupUnit, len(existingUnitGroups))
	for _, u := range allUnits {
		unitsByGroupID[u.UnitGroupID] = append(unitsByGroupID[u.UnitGroupID], u)
	}

	// If both a system group and an account-owned group share a name, prefer the
	// account-owned one so the upsert targets the right row. If only a system group
	// matched, upsertUnitGroupInTx rejects the modification because its AccountID is nil.
	unitGroupsByName := make(map[string]*domain.UnitGroupFull, len(existingUnitGroups))
	for _, unitGroup := range existingUnitGroups {
		unitGroup.UnitConversions = unitsByGroupID[unitGroup.ID]
		key := strings.ToLower(unitGroup.Name)
		current := unitGroupsByName[key]
		if current == nil || current.AccountID == nil {
			unitGroupsByName[key] = unitGroup
		}
	}

	results := make([]domain.RowResult, 0, len(rows))
	var rowErrors []apierror.RowError
	for i := range rows {
		row := rows[i]

		var upsertedID string
		var isCreate bool
		rowErr := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
			oldGroup := unitGroupsByName[names[i]]

			// A group's type is immutable, so an update validates its base unit and
			// conversions against the stored type rather than the incoming one.
			groupType := row.Type
			if oldGroup != nil {
				groupType = oldGroup.Type
			}
			if row.BaseUnitDimensionCode != groupType {
				return apierror.NewValidationErrorWithParam(
					"Base unit type does not match the unit group type.",
					fmt.Sprintf("unit_groups[%d].base_unit", i))
			}
			for j, conv := range row.Conversions {
				if conv.DimensionCode != groupType {
					return apierror.NewValidationErrorWithParam(
						"Unit type does not match the unit group type.",
						fmt.Sprintf("unit_groups[%d].unit_conversions[%d].unit", i, j))
				}
			}

			id, apiErr := upsertUnitGroupInTx(spCtx, txRepos, accountID, row, oldGroup)
			if apiErr != nil {
				return apiErr
			}
			if id == nil {
				return apierror.NewInvariantViolationError(fmt.Sprintf("problem upserting unit group, no ID exists. %+v", row))
			}
			upsertedID = *id
			isCreate = oldGroup == nil
			return nil
		})
		if rowErr != nil {
			rowErrors = append(rowErrors, apierror.NewRowError(i, rowErr))
			continue
		}

		results = append(results, newRowResult(i, upsertedID, isCreate))
	}

	return BulkWriteResult{Results: results, Errors: rowErrors, WrittenIDs: resultIDs(results)}, nil
}
