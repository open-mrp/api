package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	mediatormock "github.com/open-mrp/api/services/core-service/internal/domain/mock/mediator"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	servicemock "github.com/open-mrp/api/services/core-service/internal/domain/mock/service"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/excel"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/pagination"
)

const inventoryChangeLogExportSheet = "Inventory Change Logs"

type InventoryChangeLogExportTestSuite struct {
	suite.Suite
	svc             *inventoryChangeLogSvcImpl
	repoFactory     *factorymock.MockRepoFactory
	reportFactory   *factorymock.MockRepoFactory
	changeLogRepo   *repositorymock.MockInventoryChangeLogRepo
	accountUserRepo *repositorymock.MockAccountUserRepo
	idempotencyMed  *mediatormock.MockIdempotencyMed
	jobSvc          *servicemock.MockJobSvc
	outbox          *capturingOutboxRepo
	ctrl            *gomock.Controller
}

func (suite *InventoryChangeLogExportTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.changeLogRepo = repositorymock.NewMockInventoryChangeLogRepo(suite.ctrl)
	suite.accountUserRepo = repositorymock.NewMockAccountUserRepo(suite.ctrl)
	suite.outbox = &capturingOutboxRepo{}

	// The primary factory has no change-log repo: the walk must read through the report factory.
	suite.repoFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.repoFactory.EXPECT().NewAccountUserRepo().Return(suite.accountUserRepo).AnyTimes()
	suite.repoFactory.EXPECT().NewOutboxRepo().Return(suite.outbox).AnyTimes()
	suite.reportFactory = factorymock.NewMockRepoFactory(suite.ctrl)
	suite.reportFactory.EXPECT().NewInventoryChangeLogRepo().Return(suite.changeLogRepo).AnyTimes()

	suite.idempotencyMed = mediatormock.NewMockIdempotencyMed(suite.ctrl)
	mediatorFactory := factorymock.NewMockMediatorFactory(suite.ctrl)
	mediatorFactory.EXPECT().Build(gomock.Any()).Return(domain.Mediators{Idempotency: suite.idempotencyMed}).AnyTimes()

	suite.jobSvc = servicemock.NewMockJobSvc(suite.ctrl)
	jobSvcFactory := factorymock.NewMockJobSvcFactory(suite.ctrl)
	jobSvcFactory.EXPECT().Build(gomock.Any()).Return(suite.jobSvc).AnyTimes()

	suite.svc = NewInventoryChangeLogSvc(&InventoryChangeLogSvcConfig{
		Repos:           suite.repoFactory,
		ReportRepos:     suite.reportFactory,
		MediatorFactory: mediatorFactory,
		JobSvcFactory:   jobSvcFactory,
		TxManager:       &stubTxManager{factory: suite.repoFactory},
	}).(*inventoryChangeLogSvcImpl)
}

func (suite *InventoryChangeLogExportTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

func TestInventoryChangeLogExportTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(InventoryChangeLogExportTestSuite))
}

// renders the workbook the export worker would build for the account the caller acts for
func (suite *InventoryChangeLogExportTestSuite) build(filters domain.ExportInventoryChangeLogsParams) (*domain.Export, *apierror.APIError) {
	raw, err := json.Marshal(filters)
	suite.Require().NoError(err)
	return suite.svc.BuildExportInventoryChangeLogs(internalIdentityCtx("ac_owner"), "ac_owner", raw)
}

func changeLogPage(next *string, logs ...*domain.InventoryChangeLog) *domain.ListInventoryChangeLogsResult {
	return &domain.ListInventoryChangeLogsResult{
		Items:    logs,
		PageInfo: pagination.PageInfo{HasNextPage: next != nil, NextCursor: next},
	}
}

func scannedChangeLog() *domain.InventoryChangeLog {
	return &domain.InventoryChangeLog{
		ID:                       "inchlg_scan",
		ItemSKU:                  "YRN-RED",
		QuantityValue:            "-2.500000000000000000000000000000",
		QuantityUnitAbbreviation: "lb",
		ActionTypeCode:           string(constants.InventoryActionTypeScan),
		ResponsibleUserName:      new("Pat Lee"),
		ScanningStationName:      new("Dye House"),
		CreatedAt:                time.Date(2026, 3, 1, 15, 4, 5, 0, time.FixedZone("CST", -6*60*60)),
	}
}

func correctedChangeLog() *domain.InventoryChangeLog {
	return &domain.InventoryChangeLog{
		ID:                       "inchlg_fix",
		ItemSKU:                  "YRN-BLUE",
		QuantityValue:            "12.000000000000000000000000000000",
		QuantityUnitAbbreviation: "ea",
		ActionTypeCode:           string(constants.InventoryActionTypeUserCorrection),
		CreatedAt:                time.Date(2026, 2, 14, 9, 0, 0, 0, time.UTC),
	}
}

func (suite *InventoryChangeLogExportTestSuite) TestBuild_WritesTheSynchronousExportsColumns() {
	suite.changeLogRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(changeLogPage(nil, scannedChangeLog(), correctedChangeLog()), nil)

	export, apiErr := suite.build(domain.ExportInventoryChangeLogsParams{})
	suite.Require().Nil(apiErr)
	suite.Equal(int32(2), export.RowCount)
	suite.Equal(excel.ContentType, export.ContentType)

	rows := exportedSheetRows(suite.T(), export, inventoryChangeLogExportSheet)
	suite.Require().Len(rows, 3, "a header and two change logs")
	suite.Equal([]string{"Item", "Quantity Change", "Unit", "Action Type", "Responsible User", "Responsible Scanning Station", "Created At"}, rows[0])
	suite.Equal([]string{"YRN-RED", "-2.5", "lb", "scan", "Pat Lee", "Dye House", "2026-03-01T21:04:05Z"}, rows[1], "the amount loses its storage padding and the time reads in UTC")
	suite.Equal([]string{"YRN-BLUE", "12", "ea", "user_correction", "", "", "2026-02-14T09:00:00Z"}, rows[2])
}

func (suite *InventoryChangeLogExportTestSuite) TestBuild_EmptyWindowYieldsHeaderOnlyFile() {
	suite.changeLogRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(changeLogPage(nil), nil)

	export, apiErr := suite.build(domain.ExportInventoryChangeLogsParams{})
	suite.Require().Nil(apiErr)

	rows := exportedSheetRows(suite.T(), export, inventoryChangeLogExportSheet)
	suite.Require().Len(rows, 1)
	suite.Equal("Item", rows[0][0])
}

// The export walks the list a page at a time with the stored filters, for the caller's account whatever the stored filters say, and with no default window: an export without a start covers the whole history.
func (suite *InventoryChangeLogExportTestSuite) TestBuild_WalksTheListPageByPageWithItsFilters() {
	startsAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	stored := domain.ExportInventoryChangeLogsParams{
		AccountID:        "ac_attacker",
		ItemIDs:          []string{"itm_red"},
		ActionTypeCodes:  []string{"scan", "user_correction"},
		ChangedByUserIDs: []string{"usr_pat"},
		StartDate:        &startsAt,
	}
	next := "cur_2"
	gomock.InOrder(
		suite.changeLogRepo.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, p domain.ListInventoryChangeLogsParams) (*domain.ListInventoryChangeLogsResult, *apierror.APIError) {
				suite.Equal("ac_owner", p.AccountID, "the account is the caller's, never the stored filters'")
				suite.Nil(p.Cursor)
				suite.Equal(int32(inventoryChangeLogExportPageSize), p.Limit)
				suite.Equal(stored.ItemIDs, p.ItemIDs)
				suite.Equal(stored.ActionTypeCodes, p.ActionTypeCodes)
				suite.Equal(stored.ChangedByUserIDs, p.ChangedByUserIDs)
				suite.Require().NotNil(p.StartDate)
				suite.True(startsAt.Equal(*p.StartDate))
				suite.Nil(p.EndDate)
				suite.Nil(p.Query)
				return changeLogPage(&next, scannedChangeLog()), nil
			}),
		suite.changeLogRepo.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, p domain.ListInventoryChangeLogsParams) (*domain.ListInventoryChangeLogsResult, *apierror.APIError) {
				suite.Require().NotNil(p.Cursor)
				suite.Equal(next, *p.Cursor)
				suite.Equal(stored.ItemIDs, p.ItemIDs, "every page carries the filters")
				return changeLogPage(nil, correctedChangeLog()), nil
			}),
	)

	export, apiErr := suite.build(stored)
	suite.Require().Nil(apiErr)

	rows := exportedSheetRows(suite.T(), export, inventoryChangeLogExportSheet)
	suite.Require().Len(rows, 3)
	suite.Equal("YRN-RED", rows[1][0], "in list order")
	suite.Equal("YRN-BLUE", rows[2][0])
}

func (suite *InventoryChangeLogExportTestSuite) TestBuild_WithoutAWindowReadsNoDefaultStart() {
	suite.changeLogRepo.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, p domain.ListInventoryChangeLogsParams) (*domain.ListInventoryChangeLogsResult, *apierror.APIError) {
			suite.Nil(p.StartDate, "the list's 90-day default is not the export's")
			suite.Nil(p.EndDate)
			return changeLogPage(nil), nil
		})

	_, apiErr := suite.build(domain.ExportInventoryChangeLogsParams{})
	suite.Require().Nil(apiErr)
}

// A page that fails sinks the whole export rather than leaving a file that silently stops partway.
func (suite *InventoryChangeLogExportTestSuite) TestBuild_FailsWhenAPageCannotBeRead() {
	next := "cur_2"
	pageErr := apierror.NewInternalError(fmt.Errorf("connection reset"), "Failed to list inventory change logs.")
	gomock.InOrder(
		suite.changeLogRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(changeLogPage(&next, scannedChangeLog()), nil),
		suite.changeLogRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, pageErr),
	)

	export, apiErr := suite.build(domain.ExportInventoryChangeLogsParams{})
	suite.Nil(export)
	suite.Equal(pageErr, apiErr)
}

// The accept records only the filters, and names the file for the window so the worker and the reader agree on it.
func (suite *InventoryChangeLogExportTestSuite) TestStart_RecordsTheFiltersAndFileNameOnAnExportJob() {
	ctx := idempotencyCtx(internalIdentityCtx("ac_owner"))
	suite.accountUserRepo.EXPECT().ResolveAccountUserID(gomock.Any(), gomock.Any(), gomock.Any()).Return("acus_1", nil).AnyTimes()
	suite.idempotencyMed.EXPECT().UpsertIdempotencyKey(gomock.Any(), gomock.Any()).
		Return(&domain.IdempotencyKey{TypeID: "idk_1", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil)
	suite.idempotencyMed.EXPECT().CacheSuccessResponse(gomock.Any(), "idk_1", gomock.Any()).Return(nil)

	var recorded domain.CreateJobServiceParams
	suite.jobSvc.EXPECT().CreateJob(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, params domain.CreateJobServiceParams) (*domain.Job, *apierror.APIError) {
			recorded = params
			return &domain.Job{ID: "jb_1"}, nil
		})

	startsAt := time.Date(2026, 1, 1, 6, 0, 0, 0, time.UTC)
	job, apiErr := suite.svc.StartInventoryChangeLogsExport(ctx, domain.ExportInventoryChangeLogsParams{
		AccountID:       "ac_attacker",
		ActionTypeCodes: []string{"scan"},
		StartDate:       &startsAt,
	})
	suite.Require().Nil(apiErr)
	suite.Equal("jb_1", job.ID)
	suite.Equal(constants.JobTypeExport, recorded.Type)
	suite.Equal(constants.ObjectTypeInventoryChangeLog, recorded.ResourceType)

	var payload exportJobPayload
	suite.Require().NoError(json.Unmarshal(recorded.JobItems, &payload))
	suite.Equal("inventory_change_logs", payload.Slug)
	suite.Equal("inventory-change-logs-2026-01-01-all.xlsx", payload.Name)
	var filters domain.ExportInventoryChangeLogsParams
	suite.Require().NoError(json.Unmarshal(payload.Filters, &filters))
	suite.Empty(filters.AccountID, "the account is never stored; the worker uses the caller's")
	suite.Equal([]string{"scan"}, filters.ActionTypeCodes)
	suite.Require().NotNil(filters.StartDate)
	suite.True(startsAt.Equal(*filters.StartDate))

	suite.Require().Len(suite.outbox.messages, 1)
	suite.Equal(string(messaging.ExportInventoryChangeLogs.RoutingKey()), suite.outbox.messages[0].RoutingKey)
}

// As with the synchronous export, only the account's own staff may export its change logs, and only with inventory_logs:read.
func (suite *InventoryChangeLogExportTestSuite) TestStart_RefusesWhoeverMayNotReadInventoryLogs() {
	for name, ctx := range map[string]context.Context{
		"no identity":            context.Background(),
		"customer portal":        customerProductIdentityCtx("ac_customer", "ac_owner"),
		"no inventory_logs:read": readOnlyProductIdentityCtx("ac_owner"),
	} {
		job, apiErr := suite.svc.StartInventoryChangeLogsExport(ctx, domain.ExportInventoryChangeLogsParams{})
		suite.Nil(job, name)
		suite.NotNil(apiErr, name)
	}
}

func TestInventoryChangeLogExportFileName_NamesTheWindow(t *testing.T) {
	t.Parallel()
	at := func(s string) *time.Time {
		parsed, err := time.Parse(time.RFC3339, s)
		require.NoError(t, err)
		return &parsed
	}
	for name, tc := range map[string]struct {
		filters domain.ExportInventoryChangeLogsParams
		want    string
	}{
		"no window":                          {domain.ExportInventoryChangeLogsParams{}, "inventory-change-logs-all-all.xlsx"},
		"both bounds":                        {domain.ExportInventoryChangeLogsParams{StartDate: at("2099-01-01T00:00:00Z"), EndDate: at("2099-12-31T00:00:00Z")}, "inventory-change-logs-2099-01-01-2099-12-31.xlsx"},
		"open end":                           {domain.ExportInventoryChangeLogsParams{StartDate: at("2099-01-01T00:00:00Z")}, "inventory-change-logs-2099-01-01-all.xlsx"},
		"open start":                         {domain.ExportInventoryChangeLogsParams{EndDate: at("2099-12-31T00:00:00Z")}, "inventory-change-logs-all-2099-12-31.xlsx"},
		"offset bound reads as its UTC date": {domain.ExportInventoryChangeLogsParams{StartDate: at("2099-01-01T20:00:00-06:00")}, "inventory-change-logs-2099-01-02-all.xlsx"},
	} {
		require.Equal(t, tc.want, inventoryChangeLogExportFileName(tc.filters), name)
	}
}
