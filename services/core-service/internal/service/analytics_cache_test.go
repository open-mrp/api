package service

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/core-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/cache"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// A cache hit is a JSON round trip, so any field JSON drops would silently vanish from every cached report.
func TestAnalyticsCache_ReportTypesSurviveJSON(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeFor[[]domain.SalesEntry](),
		reflect.TypeFor[*domain.CustomerPricingAnalysis](),
		reflect.TypeFor[[]domain.OeeDepartment](),
		reflect.TypeFor[[]domain.OeeTrendPeriod](),
		reflect.TypeFor[*domain.ScheduleAttainmentResult](),
		reflect.TypeFor[*domain.DemandForecastResult](),
		reflect.TypeFor[*domain.WeeksOfSalesResult](),
		reflect.TypeFor[*domain.DeliveryPerformanceResult](),
	} {
		t.Run(typ.String(), func(t *testing.T) {
			for _, problem := range jsonLossyFields(typ, typ.String(), map[reflect.Type]bool{}) {
				t.Error(problem)
			}
		})
	}
}

var (
	jsonMarshalerType   = reflect.TypeFor[json.Marshaler]()
	jsonUnmarshalerType = reflect.TypeFor[json.Unmarshaler]()
)

func jsonLossyFields(typ reflect.Type, path string, seen map[reflect.Type]bool) []string {
	if typ.Implements(jsonMarshalerType) && reflect.PointerTo(typ).Implements(jsonUnmarshalerType) {
		return nil
	}
	switch typ.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return jsonLossyFields(typ.Elem(), path+"[]", seen)
	case reflect.Map:
		return jsonLossyFields(typ.Elem(), path+"{}", seen)
	case reflect.Interface, reflect.Func, reflect.Chan, reflect.Complex64, reflect.Complex128, reflect.UnsafePointer:
		return []string{path + ": " + typ.Kind().String() + " cannot round-trip through JSON"}
	case reflect.Struct:
		if seen[typ] {
			return nil
		}
		seen[typ] = true
		var problems []string
		for f := range typ.Fields() {
			fieldPath := path + "." + f.Name
			tag := f.Tag.Get("json")
			switch {
			case !f.IsExported():
				problems = append(problems, fieldPath+": unexported, dropped by JSON")
			case tag == "-":
				problems = append(problems, fieldPath+`: tagged json:"-", dropped by JSON`)
			case strings.Contains(tag, "omitempty") && (f.Type.Kind() == reflect.Slice || f.Type.Kind() == reflect.Map):
				problems = append(problems, fieldPath+": omitempty turns an empty collection into nil")
			default:
				problems = append(problems, jsonLossyFields(f.Type, fieldPath, seen)...)
			}
		}
		return problems
	}
	return nil
}

func TestAnalyticsCache_FamilyResourcesAreRealObjectTypes(t *testing.T) {
	for family, resources := range analyticsFamilyResources {
		for _, resource := range resources {
			require.Truef(t, resource.IsValid(), "%s lists unknown object type %q", family, resource)
		}
	}
}

func newTestAnalyticsCache(t *testing.T, now time.Time) (*AnalyticsCache, *cache.MemoryStore) {
	t.Helper()
	store, err := cache.NewMemoryStore(nil)
	require.NoError(t, err)
	c, err := NewAnalyticsCache(&AnalyticsCacheConfig{Store: store, Now: func() time.Time { return now }})
	require.NoError(t, err)
	return c, store
}

func countingOee(calls *atomic.Int32) func(context.Context) ([]domain.OeeDepartment, *apierror.APIError) {
	return func(context.Context) ([]domain.OeeDepartment, *apierror.APIError) {
		calls.Add(1)
		return []domain.OeeDepartment{}, nil
	}
}

func TestAnalyticsCache_KeysOnEffectiveParams(t *testing.T) {
	c, _ := newTestAnalyticsCache(t, time.Now())
	ctx := context.Background()
	var calls atomic.Int32
	report := func(params domain.AnalyzeOeeParams) analyticsReport {
		return analyticsReport{accountID: "ac_1", family: analyticsFamilyProduction, method: "oee", params: params}
	}
	a := domain.AnalyzeOeeParams{AccountID: "ac_1", DepartmentIDs: []string{"dp_1"}, PlannedTimeHours: map[string]float64{"dp_1": 8, "dp_2": 6}}
	b := domain.AnalyzeOeeParams{AccountID: "ac_1", DepartmentIDs: []string{"dp_2"}}

	_, _ = cachedReport(ctx, c.oee, report(a), countingOee(&calls))
	_, _ = cachedReport(ctx, c.oee, report(a), countingOee(&calls))
	_, _ = cachedReport(ctx, c.oee, report(b), countingOee(&calls))

	require.EqualValues(t, 2, calls.Load(), "identical params should share an entry; different ones should not")
}

func TestAnalyticsCache_AuditEventInvalidatesOnlyAffectedFamilyAndAccount(t *testing.T) {
	c, _ := newTestAnalyticsCache(t, time.Now())
	ctx := context.Background()
	var salesCalls, oeeCalls, otherAccountCalls atomic.Int32
	loadSales := func(context.Context) ([]domain.SalesEntry, *apierror.APIError) {
		salesCalls.Add(1)
		return []domain.SalesEntry{}, nil
	}
	sales := analyticsReport{accountID: "ac_1", family: analyticsFamilySales, method: "sales", params: 1}
	oee := analyticsReport{accountID: "ac_1", family: analyticsFamilyProduction, method: "oee", params: 1}
	otherAccount := analyticsReport{accountID: "ac_2", family: analyticsFamilySales, method: "sales", params: 1}

	warm := func() {
		_, _ = cachedReport(ctx, c.sales, sales, loadSales)
		_, _ = cachedReport(ctx, c.oee, oee, countingOee(&oeeCalls))
		_, _ = cachedReport(ctx, c.sales, otherAccount, func(context.Context) ([]domain.SalesEntry, *apierror.APIError) {
			otherAccountCalls.Add(1)
			return []domain.SalesEntry{}, nil
		})
	}
	warm()
	c.HandleAuditEvent(ctx, audit.ObservedEvent{AccountID: "ac_1", ResourceType: constants.ObjectTypeInvoice, ResourceID: "in_1", Action: constants.AuditActionUpdate})
	warm()

	require.EqualValues(t, 2, salesCalls.Load(), "an invoice edit must invalidate the account's sales reports")
	require.EqualValues(t, 1, oeeCalls.Load(), "an invoice edit must not invalidate production reports")
	require.EqualValues(t, 1, otherAccountCalls.Load(), "an invoice edit must not reach another account")

	c.HandleAuditEvent(ctx, audit.ObservedEvent{AccountID: "ac_1", ResourceType: constants.ObjectTypeUser, ResourceID: "us_1", Action: constants.AuditActionUpdate})
	warm()
	require.EqualValues(t, 2, salesCalls.Load(), "an event no report reads must not invalidate anything")
}

func TestAnalyticsCache_FlushDropsEveryAccount(t *testing.T) {
	c, _ := newTestAnalyticsCache(t, time.Now())
	ctx := context.Background()
	var calls atomic.Int32
	for range 2 {
		for _, account := range []string{"ac_1", "ac_2"} {
			_, _ = cachedReport(ctx, c.oee, analyticsReport{accountID: account, family: analyticsFamilyProduction, method: "oee", params: 1}, countingOee(&calls))
		}
		c.Flush()
	}
	require.EqualValues(t, 4, calls.Load())
}

func TestAnalyticsCache_ClosedWindowsLiveLonger(t *testing.T) {
	now := time.Date(2026, 9, 24, 15, 0, 0, 0, time.UTC)
	c, _ := newTestAnalyticsCache(t, now)

	require.Equal(t, c.cfg.ClosedTTL, c.ttlForWindow(time.Date(2026, 9, 22, 23, 59, 0, 0, time.UTC)))
	require.Equal(t, c.cfg.LiveTTL, c.ttlForWindow(time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC)))
	require.Equal(t, c.cfg.LiveTTL, c.ttlForWindow(now.AddDate(0, 0, 7)))
}

func TestAnalyticsCache_DisabledWithoutStore(t *testing.T) {
	c, err := NewAnalyticsCache(nil)
	require.NoError(t, err)
	var calls atomic.Int32
	for range 2 {
		_, _ = cachedReport(context.Background(), c.oee, analyticsReport{accountID: "ac_1", family: analyticsFamilyProduction, method: "oee", params: 1}, countingOee(&calls))
	}
	c.HandleAuditEvent(context.Background(), audit.ObservedEvent{AccountID: "ac_1", ResourceType: constants.ObjectTypeBatch})
	c.Flush()
	require.EqualValues(t, 2, calls.Load())
}

// --- AnalyzeSales through the service: authorization and caller scoping ---

func salesCtx(accountID, roleType string, permissions map[string]bool) context.Context {
	return appctx.WithIdentity(context.Background(), &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: accountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           "us_" + roleType,
			AccountID:    &accountID,
			RoleType:     &roleType,
			Permissions:  permissions,
		},
	})
}

type salesFixture struct {
	svc         *analyticsSvcImpl
	analytics   *repositorymock.MockAnalyticsRepo
	accountUser *repositorymock.MockAccountUserRepo
	window      domain.AnalyzeSalesParams
}

func newSalesFixture(t *testing.T) *salesFixture {
	t.Helper()
	ctrl := gomock.NewController(t)
	analytics := repositorymock.NewMockAnalyticsRepo(ctrl)
	accountUser := repositorymock.NewMockAccountUserRepo(ctrl)
	repos := factorymock.NewMockRepoFactory(ctrl)
	repos.EXPECT().NewAnalyticsRepo().Return(analytics).AnyTimes()
	repos.EXPECT().NewAccountUserRepo().Return(accountUser).AnyTimes()

	reportCache, _ := newTestAnalyticsCache(t, time.Now())
	return &salesFixture{
		svc:         &analyticsSvcImpl{repos: repos, cache: reportCache},
		analytics:   analytics,
		accountUser: accountUser,
		window: domain.AnalyzeSalesParams{
			StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
		},
	}
}

func TestAnalyzeSales_CachedResultStillRequiresPermission(t *testing.T) {
	f := newSalesFixture(t)
	f.analytics.EXPECT().GetSalesEntries(gomock.Any(), gomock.Any()).Return([]domain.SalesEntry{{UnitCost: 4}}, nil).Times(1)

	allowed := salesCtx("ac_1", string(constants.RoleTypeCustom), map[string]bool{"invoices:read": true})
	for range 2 {
		entries, apiErr := f.svc.AnalyzeSales(allowed, f.window)
		require.Nil(t, apiErr)
		require.Len(t, entries, 1)
	}

	denied := salesCtx("ac_1", string(constants.RoleTypeCustom), map[string]bool{"orders:read": true})
	entries, apiErr := f.svc.AnalyzeSales(denied, f.window)
	require.NotNil(t, apiErr, "a warm cache must not bypass the permission check")
	require.Nil(t, entries)
}

func TestAnalyzeSales_SalesRepNeverSharesAnotherCallersEntry(t *testing.T) {
	f := newSalesFixture(t)
	f.accountUser.EXPECT().FindByAccountAndUserID(gomock.Any(), gomock.Any(), "ac_1").Return(&domain.AccountUser{ID: "acus_rep"}, nil).AnyTimes()
	f.analytics.EXPECT().GetSalesEntries(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, params domain.AnalyzeSalesParams) ([]domain.SalesEntry, *apierror.APIError) {
		if len(params.SalesRepIDs) == 1 && params.SalesRepIDs[0] == "acus_rep" {
			return []domain.SalesEntry{{UnitCost: 7}}, nil
		}
		return []domain.SalesEntry{{UnitCost: 7}, {UnitCost: 9}}, nil
	}).Times(2)

	admin := salesCtx("ac_1", string(constants.RoleTypeAdmin), nil)
	rep := salesCtx("ac_1", string(constants.RoleTypeSalesRep), map[string]bool{"invoices:read": true})

	adminEntries, apiErr := f.svc.AnalyzeSales(admin, f.window)
	require.Nil(t, apiErr)
	require.Len(t, adminEntries, 2)

	repEntries, apiErr := f.svc.AnalyzeSales(rep, f.window)
	require.Nil(t, apiErr)
	require.Len(t, repEntries, 1, "a sales rep must get only their own sales, not the admin's cached entry")
	require.Zero(t, repEntries[0].UnitCost, "a sales rep must not see cost")

	adminEntries, apiErr = f.svc.AnalyzeSales(admin, f.window)
	require.Nil(t, apiErr)
	require.EqualValues(t, 7, adminEntries[0].UnitCost, "zeroing a rep's copy must not touch the cached entry")
}
