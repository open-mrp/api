package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

type orderBookFixture struct {
	svc         *analyticsSvcImpl
	analytics   *repositorymock.MockAnalyticsRepo
	accountUser *repositorymock.MockAccountUserRepo
	cache       *AnalyticsCache
}

func newOrderBookFixture(t *testing.T, now time.Time) *orderBookFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	analytics := repositorymock.NewMockAnalyticsRepo(ctrl)
	accountUser := repositorymock.NewMockAccountUserRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewAnalyticsRepo().Return(analytics).AnyTimes()
	repos.EXPECT().NewAccountUserRepo().Return(accountUser).AnyTimes()
	accountUser.EXPECT().FindByAccountAndUserID(gomock.Any(), gomock.Any(), "ac_1").Return(&domain.AccountUser{ID: "acus_rep"}, nil).AnyTimes()

	reportCache, _ := newTestAnalyticsCache(t, now)
	return &orderBookFixture{
		svc:         &analyticsSvcImpl{repos: repos, cache: reportCache},
		analytics:   analytics,
		accountUser: accountUser,
		cache:       reportCache,
	}
}

func TestQuarterlyOrdersIssuedFrom(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 5, 23, 30, 0, 0, time.FixedZone("west", -8*3600))
	require.Equal(t, time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), quarterlyOrdersIssuedFrom(now, 5), "the current year counts as one; years are UTC")
	require.Equal(t, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), quarterlyOrdersIssuedFrom(now, 1))
}

func TestAnalyzeQuarterlyOrders_BoundsYearsScopesRepsAndCaches(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	f := newOrderBookFixture(t, now)

	var seen []domain.AnalyzeQuarterlyOrdersParams
	f.analytics.EXPECT().GetQuarterlyOrders(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, p domain.AnalyzeQuarterlyOrdersParams) ([]domain.YearlyQuarterlyData, *apierror.APIError) {
		seen = append(seen, p)
		return []domain.YearlyQuarterlyData{{Year: 2026, Data: domain.QuarterlyData{Q1: 1, Total: 1}}}, nil
	}).Times(3)

	admin := salesCtx("ac_1", string(constants.RoleTypeAdmin), nil)
	rep := salesCtx("ac_1", string(constants.RoleTypeSalesRep), map[string]bool{"invoices:read": true})

	_, apiErr := f.svc.AnalyzeQuarterlyOrders(admin, domain.AnalyzeQuarterlyOrdersParams{SalesRepIDs: []string{"acus_other"}})
	require.Nil(t, apiErr)
	_, apiErr = f.svc.AnalyzeQuarterlyOrders(admin, domain.AnalyzeQuarterlyOrdersParams{SalesRepIDs: []string{"acus_other"}})
	require.Nil(t, apiErr, "the second identical request is served from the cache")
	_, apiErr = f.svc.AnalyzeQuarterlyOrders(rep, domain.AnalyzeQuarterlyOrdersParams{SalesRepIDs: []string{"acus_other"}, YearsBack: 2})
	require.Nil(t, apiErr)

	require.Len(t, seen, 2)
	require.Equal(t, "ac_1", seen[0].AccountID)
	require.EqualValues(t, 5, seen[0].YearsBack)
	require.Equal(t, time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC), seen[0].IssuedFrom)
	require.Equal(t, []string{"acus_other"}, seen[0].SalesRepIDs)
	require.Equal(t, []string{"acus_rep"}, seen[1].SalesRepIDs, "a sales rep sees only their own orders, whatever they ask for")
	require.Equal(t, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), seen[1].IssuedFrom)

	// An order edit drops the cached report.
	f.cache.HandleAuditEvent(context.Background(), audit.ObservedEvent{AccountID: "ac_1", ResourceType: constants.ObjectTypeSalesOrder})
	_, apiErr = f.svc.AnalyzeQuarterlyOrders(admin, domain.AnalyzeQuarterlyOrdersParams{SalesRepIDs: []string{"acus_other"}})
	require.Nil(t, apiErr)
	require.Len(t, seen, 3)

	_, apiErr = f.svc.AnalyzeQuarterlyOrders(admin, domain.AnalyzeQuarterlyOrdersParams{YearsBack: -1})
	require.NotNil(t, apiErr)
	require.Equal(t, "years_back", apiErr.Param)
}

func TestOpenOrders_ScopeSalesRepsAndSplitPermissions(t *testing.T) {
	t.Parallel()
	f := newOrderBookFixture(t, time.Now())

	var filters []domain.OpenOrderFilter
	f.analytics.EXPECT().GetOpenOrdersSummary(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, filter domain.OpenOrderFilter) (*domain.OpenOrdersSummary, *apierror.APIError) {
		filters = append(filters, filter)
		return &domain.OpenOrdersSummary{Ordered: "1", BackOrdered: "1", Invoiced: "0"}, nil
	}).Times(2)

	invoices := salesCtx("ac_1", string(constants.RoleTypeCustom), map[string]bool{"invoices:read": true})
	rep := salesCtx("ac_1", string(constants.RoleTypeSalesRep), map[string]bool{"invoices:read": true, "sales_orders:read": true})

	_, apiErr := f.svc.AnalyzeOpenOrdersSummary(invoices, domain.OpenOrderFilter{SalesRepIDs: []string{"acus_x"}, ItemIDs: []string{"it_1"}})
	require.Nil(t, apiErr)
	_, apiErr = f.svc.AnalyzeOpenOrdersSummary(rep, domain.OpenOrderFilter{SalesRepIDs: []string{"acus_x"}, ItemIDs: []string{"it_1"}})
	require.Nil(t, apiErr, "a sales rep's forced filter is a different cache entry")
	require.Equal(t, domain.OpenOrderFilter{AccountID: "ac_1", SalesRepIDs: []string{"acus_x"}, ItemIDs: []string{"it_1"}}, filters[0])
	require.Equal(t, domain.OpenOrderFilter{AccountID: "ac_1", SalesRepIDs: []string{"acus_rep"}, ItemIDs: []string{"it_1"}}, filters[1])

	// The order detail is gated on sales orders, the totals on invoices, as the dashboard gated them.
	_, apiErr = f.svc.ListOpenOrderLines(invoices, "or_1")
	require.NotNil(t, apiErr)
	require.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code)

	salesOrders := salesCtx("ac_1", string(constants.RoleTypeCustom), map[string]bool{"sales_orders:read": true})
	_, apiErr = f.svc.AnalyzeOpenOrdersSummary(salesOrders, domain.OpenOrderFilter{})
	require.NotNil(t, apiErr)
	require.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

func TestListOpenOrderLines_ScopesRepsAndReportsAMissingOrder(t *testing.T) {
	t.Parallel()
	f := newOrderBookFixture(t, time.Now())

	repID := "acus_rep"
	f.analytics.EXPECT().GetOpenOrderLines(gomock.Any(), "ac_1", "or_theirs", &repID).Return(nil, false, nil)
	f.analytics.EXPECT().GetOpenOrderLines(gomock.Any(), "ac_1", "or_1", (*string)(nil)).Return([]domain.OpenOrderLine{{ID: "orln_1"}}, true, nil)

	rep := salesCtx("ac_1", string(constants.RoleTypeSalesRep), map[string]bool{"sales_orders:read": true})
	_, apiErr := f.svc.ListOpenOrderLines(rep, "or_theirs")
	require.NotNil(t, apiErr)
	require.Equal(t, apierror.ErrorCodeResourceNotFound, apiErr.Code)

	admin := salesCtx("ac_1", string(constants.RoleTypeAdmin), nil)
	lines, apiErr := f.svc.ListOpenOrderLines(admin, "or_1")
	require.Nil(t, apiErr)
	require.Len(t, lines, 1)
}

func TestOpenOrderLinesExport_HidesCostFromSalesReps(t *testing.T) {
	t.Parallel()
	spec := (&analyticsSvcImpl{}).openOrderLinesExportSpec()

	entry := domain.OrderEntry{OrderNumber: "000042", UnitCost: 3, UnitPrice: 5}
	shown := spec.ColumnsFor([]openOrderLineExportRow{{entry: entry}})
	hidden := spec.ColumnsFor([]openOrderLineExportRow{{entry: entry, hideCost: true}})
	require.Len(t, shown, len(openOrderLineExportColumns))
	require.Len(t, hidden, len(openOrderLineExportColumns)-1)
	for _, c := range hidden {
		require.NotEqual(t, "unitCost", c.Key)
	}

	require.Equal(t, 3.0, spec.Project(openOrderLineExportRow{entry: entry})["unitCost"])
	_, has := spec.Project(openOrderLineExportRow{entry: entry, hideCost: true})["unitCost"]
	require.False(t, has)
}

func TestClampOpenOrderLimit(t *testing.T) {
	t.Parallel()
	require.EqualValues(t, 10, clampOpenOrderLimit(0))
	require.EqualValues(t, 25, clampOpenOrderLimit(25))
	require.EqualValues(t, 100, clampOpenOrderLimit(1000))
}

// An item first ordered this month has no complete month; the dashboard still drew its forecast at zero, and a chart with no points is hidden.
func TestSeasonalEMAForecast_EmptyHistoryForecastsZero(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	points := seasonalEMAForecast(nil, 3, base, 1, func(fm forecastMonth) float64 { return fm.demand })
	require.Equal(t, []domain.DemandForecastPoint{
		{Date: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)},
		{Date: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
	}, points)
}
