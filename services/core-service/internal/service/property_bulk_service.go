package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/excel"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/messaging"
)

// hands the async bulk engine the plumbing it runs on
func (s *propertySvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

// rejects a name or value claimed twice in one request, case-insensitively. A collision
// with a stored value needs the DB, so the write reports that one.
func validateBulkUpsertPropertyRows(rows []domain.UpsertPropertyParams) *apierror.APIError {
	nameInputSpace := make(map[string]struct{}, len(rows))
	// One value belongs to one property per account, so the first row to claim it owns it.
	propertyByValue := make(map[string]string)
	var rowErrs apierror.RowErrors

	for i, property := range rows {
		lowerName := strings.ToLower(strings.TrimSpace(property.Name))
		if _, dup := nameInputSpace[lowerName]; dup {
			rowErrs.AddValidation(i, fmt.Sprintf("properties[%d].name", i), fmt.Sprintf("duplicate name %q in request", property.Name))
		}
		nameInputSpace[lowerName] = struct{}{}

		valueInputSpace := make(map[string]struct{}, len(property.Attributes))
		for j, attribute := range property.Attributes {
			param := fmt.Sprintf("properties[%d].attributes[%d].value", i, j)
			value := strings.TrimSpace(attribute.Value)
			if value == "" {
				rowErrs.AddValidation(i, param, "value cannot be blank")
				continue
			}

			lowerValue := strings.ToLower(value)
			if _, dup := valueInputSpace[lowerValue]; dup {
				rowErrs.AddValidation(i, param, fmt.Sprintf("duplicate value %q in property %q", attribute.Value, property.Name))
				continue
			}
			valueInputSpace[lowerValue] = struct{}{}

			if owner, claimed := propertyByValue[lowerValue]; claimed && !strings.EqualFold(owner, property.Name) {
				rowErrs.AddValidation(i, param, fmt.Sprintf("value %q is already used under property %q in this request", attribute.Value, owner))
				continue
			}
			propertyByValue[lowerValue] = property.Name
		}
	}

	return rowErrs.Summary("properties")
}

// trims each row and settles every swatch, so the job records a determined write.
// A property references nothing, so this reads no database.
func resolveBulkUpsertPropertyRows(_ context.Context, _ domain.RepoFactory, _ string, rows []domain.UpsertPropertyParams) ([]domain.ResolvedUpsertPropertyRow, *apierror.APIError) {
	resolved := make([]domain.ResolvedUpsertPropertyRow, len(rows))
	for i, property := range rows {
		name := strings.TrimSpace(property.Name)

		attributes := make([]domain.ResolvedUpsertPropertyAttribute, 0, len(property.Attributes))
		for _, attribute := range property.Attributes {
			value := strings.TrimSpace(attribute.Value)
			colorCode := attributeColorFor(name, value)
			if attribute.ColorCode != nil {
				colorCode = *attribute.ColorCode
			}
			attributes = append(attributes, domain.ResolvedUpsertPropertyAttribute{Value: value, ColorCode: colorCode})
		}

		resolved[i] = domain.ResolvedUpsertPropertyRow{Name: name, Attributes: attributes}
	}
	return resolved, nil
}

// upserts each row in its own savepoint, matched by name, so a bad row rolls back alone
func writeBulkUpsertProperties(txCtx context.Context, txRepos domain.RepoFactory, sp db.SavepointRunner, accountID string, rows []domain.ResolvedUpsertPropertyRow) (BulkWriteResult, *apierror.APIError) {
	existing, apiErr := loadBulkUpsertPropertyState(txCtx, txRepos, accountID, rows)
	if apiErr != nil {
		return BulkWriteResult{}, apiErr
	}

	results := make([]domain.RowResult, 0, len(rows))
	var rowErrors []apierror.RowError
	for i := range rows {
		row := rows[i]

		var upsertedID string
		var isCreate bool
		rowErr := sp.Run(txCtx, func(spCtx context.Context) *apierror.APIError {
			old := existing.propertyByName[strings.ToLower(row.Name)]

			propertyID, apiErr := upsertPropertyInTx(spCtx, txRepos, accountID, row, old)
			if apiErr != nil {
				return apiErr
			}
			if apiErr := writeRowAttributesInTx(spCtx, txRepos, accountID, i, propertyID, row, existing); apiErr != nil {
				return apiErr
			}

			upsertedID = propertyID
			isCreate = old == nil
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

// carries the batch's one-time reads. Mutated as rows commit, so a value an earlier row
// created still conflicts with a later one.
type bulkUpsertPropertyState struct {
	// the row to upsert, keyed by lower-cased name.
	propertyByName map[string]*domain.Property
	// values already defined, keyed by property id then lower-cased value.
	attributeValuesByProperty map[string]map[string]struct{}
	// gives each new attribute its next sort order.
	attributeCountByProperty map[string]int32
	// owns the account-wide value uniqueness check, keyed by lower-cased value.
	propertyNameByValue map[string]string
}

// reads what the rows are matched and checked against, three queries for the whole batch
func loadBulkUpsertPropertyState(txCtx context.Context, txRepos domain.RepoFactory, accountID string, rows []domain.ResolvedUpsertPropertyRow) (*bulkUpsertPropertyState, *apierror.APIError) {
	names := make([]string, len(rows))
	for i, row := range rows {
		names[i] = strings.ToLower(row.Name)
	}

	existing, apiErr := txRepos.NewPropertyRepo().FindByNames(txCtx, accountID, names)
	if apiErr != nil {
		return nil, apiErr
	}
	propertyByName := make(map[string]*domain.Property, len(existing))
	propertyIDs := make([]string, 0, len(existing))
	for _, property := range existing {
		propertyByName[strings.ToLower(property.Name)] = property
		propertyIDs = append(propertyIDs, property.ID)
	}

	attributeRepo := txRepos.NewAttributeRepo()
	attributeValuesByProperty := make(map[string]map[string]struct{}, len(propertyIDs))
	attributeCountByProperty := make(map[string]int32, len(propertyIDs))
	if len(propertyIDs) > 0 {
		attributes, apiErr := attributeRepo.ListByPropertyIDs(txCtx, accountID, propertyIDs)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, attribute := range attributes {
			values := attributeValuesByProperty[attribute.PropertyID]
			if values == nil {
				values = map[string]struct{}{}
				attributeValuesByProperty[attribute.PropertyID] = values
			}
			values[strings.ToLower(strings.TrimSpace(attribute.Value))] = struct{}{}
			attributeCountByProperty[attribute.PropertyID]++
		}
	}

	var incomingValues []string
	for _, row := range rows {
		for _, attribute := range row.Attributes {
			incomingValues = append(incomingValues, attribute.Value)
		}
	}
	propertyNameByValue := make(map[string]string, len(incomingValues))
	if len(incomingValues) > 0 {
		matches, apiErr := attributeRepo.FindByTextsInAccount(txCtx, accountID, incomingValues)
		if apiErr != nil {
			return nil, apiErr
		}
		for _, match := range matches {
			propertyNameByValue[strings.ToLower(strings.TrimSpace(match.Text))] = match.PropertyName
		}
	}

	return &bulkUpsertPropertyState{
		propertyByName:            propertyByName,
		attributeValuesByProperty: attributeValuesByProperty,
		attributeCountByProperty:  attributeCountByProperty,
		propertyNameByValue:       propertyNameByValue,
	}, nil
}

// creates one property, or renames an existing one to the request's casing
func upsertPropertyInTx(txCtx context.Context, txRepos domain.RepoFactory, accountID string, row domain.ResolvedUpsertPropertyRow, old *domain.Property) (string, *apierror.APIError) {
	ctx, span := propertySvcTracer.Start(txCtx, "service.property.upsert_in_tx")
	defer span.End()

	txRepo := txRepos.NewPropertyRepo()

	if old == nil {
		propertyID, apiErr := id.GenID(id.PropertyIDPrefix, nil)
		if apiErr != nil {
			return "", apiErr
		}

		created, apiErr := txRepo.Create(ctx, propertyID, domain.CreatePropertyParams{
			AccountID: accountID,
			Name:      row.Name,
		})
		if apiErr != nil {
			return "", apiErr
		}

		if apiErr := publishPropertyAudit(ctx, txRepos, constants.AuditActionCreate, created.ID, audit.ComputeChanges(nil, created)); apiErr != nil {
			return "", apiErr
		}
		return created.ID, nil
	}

	name := row.Name
	updated, apiErr := txRepo.Update(ctx, domain.UpdatePropertyParams{
		PropertyID: old.ID,
		AccountID:  accountID,
		Name:       &name,
	})
	if apiErr != nil {
		return "", apiErr
	}

	if apiErr := publishPropertyAudit(ctx, txRepos, constants.AuditActionUpdate, updated.ID, audit.ComputeChanges(old, updated)); apiErr != nil {
		return "", apiErr
	}
	return updated.ID, nil
}

// adds only the values the property does not already carry, so a re-import is a no-op
func writeRowAttributesInTx(txCtx context.Context, txRepos domain.RepoFactory, accountID string, rowIndex int, propertyID string, row domain.ResolvedUpsertPropertyRow, state *bulkUpsertPropertyState) *apierror.APIError {
	values := state.attributeValuesByProperty[propertyID]
	if values == nil {
		values = map[string]struct{}{}
		state.attributeValuesByProperty[propertyID] = values
	}

	attributeRepo := txRepos.NewAttributeRepo()
	for j, attribute := range row.Attributes {
		lowerValue := strings.ToLower(attribute.Value)
		if _, defined := values[lowerValue]; defined {
			continue
		}

		// One value belongs to one property per account; the single create path 409s too.
		if owner, taken := state.propertyNameByValue[lowerValue]; taken && !strings.EqualFold(owner, row.Name) {
			return apierror.NewConflictErrorWithParam(
				fmt.Sprintf("Attribute value %q already exists under property %q.", attribute.Value, owner),
				fmt.Sprintf("properties[%d].attributes[%d].value", rowIndex, j))
		}

		attributeID, apiErr := id.GenID(id.AttributeIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		state.attributeCountByProperty[propertyID]++
		created, apiErr := attributeRepo.Create(txCtx, attributeID, domain.CreateAttributeParams{
			Value:      attribute.Value,
			PropertyID: propertyID,
			AccountID:  accountID,
			ColorCode:  attribute.ColorCode,
			SortOrder:  state.attributeCountByProperty[propertyID],
		})
		if apiErr != nil {
			return apiErr
		}

		if apiErr := audit.NewPublisher().Publish(txCtx, txRepos.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeAttribute,
			ResourceID:   created.ID,
			Changes:      audit.ComputeChanges(nil, created),
		}); apiErr != nil {
			return apiErr
		}

		values[lowerValue] = struct{}{}
		state.propertyNameByValue[lowerValue] = row.Name
	}

	return nil
}

// records one property change on the outbox
func publishPropertyAudit(txCtx context.Context, txRepos domain.RepoFactory, action constants.AuditAction, propertyID string, changes []audit.FieldChange) *apierror.APIError {
	return audit.NewPublisher().Publish(txCtx, txRepos.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       action,
		ResourceType: constants.ObjectTypeProperty,
		ResourceID:   propertyID,
		Changes:      changes,
	})
}

// wires properties into the async bulk engine
func (s *propertySvcImpl) bulkUpsertSpec() bulkOperationSpec[domain.UpsertPropertyParams, domain.ResolvedUpsertPropertyRow] {
	return bulkOperationSpec[domain.UpsertPropertyParams, domain.ResolvedUpsertPropertyRow]{
		JobType:          constants.JobTypeBulkUpsert,
		RoutingKey:       messaging.BulkUpsertProperties.RoutingKey(),
		PermissionDomain: types.PermissionDomainProperties,
		Actions:          []types.Action{types.ActionCreate, types.ActionUpdate},
		EntityName:       "properties",
		Validate:         validateBulkUpsertPropertyRows,
		Resolve:          resolveBulkUpsertPropertyRows,
		Write:            writeBulkUpsertProperties,
	}
}

// accepts a bulk upsert: validates synchronously, records the rows on a job to poll
func (s *propertySvcImpl) BulkUpsertProperties(ctx context.Context, params domain.BulkUpsertPropertiesParams) (*domain.Job, *apierror.APIError) {
	return enqueueBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), params.Properties)
}

// writes an enqueued bulk upsert. At-least-once delivery, made effectively-once by the
// inbox de-dup and the engine's terminal-job guard.
func (s *propertySvcImpl) ExecuteBulkUpsertProperties(ctx context.Context, event domain.BulkOperationJobEvent) *apierror.APIError {
	return executeBulkOperation(ctx, s.asyncBulkDeps(), s.bulkUpsertSpec(), event)
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

// wires properties into the export engine, one attribute per row under its property
func (s *propertySvcImpl) exportSpec() exportSpec[*domain.Property, domain.ExportPropertiesParams] {
	return exportSpec[*domain.Property, domain.ExportPropertiesParams]{
		PermissionDomain: types.PermissionDomainProperties,
		Name:             "Properties",
		Slug:             "properties",
		Columns: []excel.ColumnSpec{
			{Header: "ID", Key: "id", Width: 20},
			{Header: "Name", Key: "name", Width: 28},
			{
				Header: "Attribute", Key: "attribute", Width: 28,
				Note: `OPTIONAL. One selectable value per row, listed under its property. Values are created if they do not already exist, and must be unique across the account.`,
			},
			{
				Header: "Color", Key: "color", Width: 14,
				Note: `OPTIONAL. Swatch color for the attribute on the same row. One of: blue, brown, gray, green, orange, pink, purple, red, yellow, default. Assigned automatically when blank.`,
			},
		},

		Fetch: func(ctx context.Context, repos domain.RepoFactory, accountID string, filters domain.ExportPropertiesParams) ([]*domain.Property, *apierror.APIError) {
			filters.AccountID = accountID
			return fetchPropertiesWithAttributes(ctx, repos, filters)
		},

		Expand: func(property *domain.Property) []excel.Row {
			parent := excel.Row{
				"id":   property.ID,
				"name": property.Name,
			}

			children := make([]excel.Row, 0, len(property.Attributes))
			for _, attribute := range property.Attributes {
				children = append(children, excel.Row{
					"attribute": attribute.Value,
					"color":     attribute.ColorCode,
				})
			}

			return excel.Group(parent, children)
		},
	}
}

// reads the matching properties with their attributes, in the order the sheet lists them
func fetchPropertiesWithAttributes(ctx context.Context, repos domain.RepoFactory, filters domain.ExportPropertiesParams) ([]*domain.Property, *apierror.APIError) {
	properties, apiErr := repos.NewPropertyRepo().Export(ctx, filters)
	if apiErr != nil {
		return nil, apiErr
	}
	if len(properties) == 0 {
		return properties, nil
	}

	propertyIDs := make([]string, len(properties))
	for i, property := range properties {
		propertyIDs[i] = property.ID
	}
	attributes, apiErr := repos.NewAttributeRepo().ListByPropertyIDs(ctx, filters.AccountID, propertyIDs)
	if apiErr != nil {
		return nil, apiErr
	}

	byProperty := make(map[string][]*domain.Attribute, len(properties))
	for _, attribute := range attributes {
		byProperty[attribute.PropertyID] = append(byProperty[attribute.PropertyID], attribute)
	}
	for _, property := range properties {
		property.Attributes = byProperty[property.ID]
	}

	return properties, nil
}

// accepts an export: records what to build on a job to poll
func (s *propertySvcImpl) ExportProperties(ctx context.Context, params domain.ExportPropertiesParams) (*domain.Job, *apierror.APIError) {
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), params)
}

// renders the file an accepted export recorded
func (s *propertySvcImpl) BuildExportProperties(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.repos, s.exportSpec())(ctx, accountID, filters)
}

// prefixes the column key of a category property, keeping it clear of the fixed
// item columns
const propertyKeyPrefix = "property:"

// derives one column per distinct property name across the exported items'
// categories. Name, not id, is the identity: that is what the importer reads back.
func itemPropertyColumns(items []*domain.Item) []excel.ColumnSpec {
	seen := map[string]struct{}{}
	names := []string{}
	for _, item := range items {
		if item == nil || item.Category == nil {
			continue
		}
		for _, property := range item.Category.Properties {
			name := strings.TrimSpace(property.Name)
			if name == "" {
				continue
			}
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}

	// Sorted, because first-encounter order depends on which rows came back and
	// would reshuffle the sheet between exports of the same data.
	sort.Strings(names)

	columns := make([]excel.ColumnSpec, len(names))
	for i, name := range names {
		columns[i] = excel.ColumnSpec{Header: name, Key: propertyKeyPrefix + name, Width: 15}
	}
	return columns
}

// writes an item's attribute values into its row, keyed to match the property
// columns above
func addItemPropertyCells(row excel.Row, item *domain.Item) {
	if item == nil || item.Category == nil {
		return
	}

	nameByPropertyID := make(map[string]string, len(item.Category.Properties))
	for _, property := range item.Category.Properties {
		nameByPropertyID[property.ID] = strings.TrimSpace(property.Name)
	}

	for _, attribute := range item.Attributes {
		if attribute == nil {
			continue
		}
		if name, ok := nameByPropertyID[attribute.PropertyID]; ok && name != "" {
			row[propertyKeyPrefix+name] = attribute.Value
		}
	}
}
