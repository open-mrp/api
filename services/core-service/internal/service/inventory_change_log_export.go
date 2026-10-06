package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/excel"
)

// inventoryChangeLogExportPageSize keeps each read of the walk well inside the query budget, even under a filter naming many users or items, whose page costs several times an unfiltered one.
const inventoryChangeLogExportPageSize = 250

// inventoryChangeLogExportColumns are the synchronous export's columns, in its order.
var inventoryChangeLogExportColumns = []excel.ColumnSpec{
	{Header: "Item", Key: "item"},
	{Header: "Quantity Change", Key: "quantityChange"},
	{Header: "Unit", Key: "unit"},
	{Header: "Action Type", Key: "actionType"},
	{Header: "Responsible User", Key: "responsibleUser"},
	{Header: "Responsible Scanning Station", Key: "responsibleScanningStation"},
	{Header: "Created At", Key: "createdAt"},
}

func (s *inventoryChangeLogSvcImpl) asyncBulkDeps() asyncBulkDeps {
	return asyncBulkDeps{
		repos:           s.repos,
		mediatorFactory: s.mediatorFactory,
		jobSvcFactory:   s.jobSvcFactory,
		txManager:       s.txManager,
	}
}

func (s *inventoryChangeLogSvcImpl) exportSpec() exportSpec[*domain.InventoryChangeLog, domain.ExportInventoryChangeLogsParams] {
	return exportSpec[*domain.InventoryChangeLog, domain.ExportInventoryChangeLogsParams]{
		PermissionDomain: types.PermissionDomainInventoryLogs,
		Name:             "Inventory Change Logs",
		Slug:             "inventory_change_logs",
		ResourceType:     constants.ObjectTypeInventoryChangeLog,
		Columns:          inventoryChangeLogExportColumns,
		Stream:           streamInventoryChangeLogExportRows,
		FileName:         inventoryChangeLogExportFileName,
		Project:          projectInventoryChangeLogExportRow,
	}
}

// StartInventoryChangeLogsExport stores only the filters; the account is the caller's when the file is built.
func (s *inventoryChangeLogSvcImpl) StartInventoryChangeLogsExport(ctx context.Context, filters domain.ExportInventoryChangeLogsParams) (*domain.Job, *apierror.APIError) {
	filters.AccountID = ""
	return enqueueExport(ctx, s.asyncBulkDeps(), s.exportSpec(), filters)
}

func (s *inventoryChangeLogSvcImpl) BuildExportInventoryChangeLogs(ctx context.Context, accountID string, filters json.RawMessage) (*domain.Export, *apierror.APIError) {
	return exportBuilder(s.reportRepos, s.exportSpec())(ctx, accountID, filters)
}

// streamInventoryChangeLogExportRows walks the change-log list a page at a time, newest first, so the file holds what the list shows for the same filters. Unlike the list, no default window applies: an export without a start covers the account's whole history.
func streamInventoryChangeLogExportRows(
	ctx context.Context,
	repos domain.RepoFactory,
	accountID string,
	filters domain.ExportInventoryChangeLogsParams,
	emit func([]*domain.InventoryChangeLog) *apierror.APIError,
) *apierror.APIError {
	logs := repos.NewInventoryChangeLogRepo()
	params := domain.ListInventoryChangeLogsParams{
		AccountID:        accountID,
		Limit:            inventoryChangeLogExportPageSize,
		ItemIDs:          filters.ItemIDs,
		ActionTypeCodes:  filters.ActionTypeCodes,
		ChangedByUserIDs: filters.ChangedByUserIDs,
		StartDate:        filters.StartDate,
		EndDate:          filters.EndDate,
	}
	for {
		page, apiErr := logs.List(ctx, params)
		if apiErr != nil {
			return apiErr
		}
		if apiErr := emit(page.Items); apiErr != nil {
			return apiErr
		}
		if !page.PageInfo.HasNextPage || page.PageInfo.NextCursor == nil {
			return nil
		}
		params.Cursor = page.PageInfo.NextCursor
	}
}

// inventoryChangeLogExportFileName names the file for the window it covers, each bound by its UTC date and an open one as `all`, so two ranges downloaded side by side do not overwrite each other.
func inventoryChangeLogExportFileName(filters domain.ExportInventoryChangeLogsParams) string {
	return "inventory-change-logs-" + inventoryChangeLogWindowBound(filters.StartDate) + "-" + inventoryChangeLogWindowBound(filters.EndDate) + ".xlsx"
}

func inventoryChangeLogWindowBound(at *time.Time) string {
	if at == nil {
		return "all"
	}
	return at.UTC().Format(time.DateOnly)
}

func projectInventoryChangeLogExportRow(icl *domain.InventoryChangeLog) excel.Row {
	return excel.Row{
		"item":                       icl.ItemSKU,
		"quantityChange":             decimalCell(icl.QuantityValue),
		"unit":                       icl.QuantityUnitAbbreviation,
		"actionType":                 icl.ActionTypeCode,
		"responsibleUser":            excel.Str(icl.ResponsibleUserName),
		"responsibleScanningStation": excel.Str(icl.ScanningStationName),
		"createdAt":                  icl.CreatedAt.UTC().Format(time.RFC3339),
	}
}
