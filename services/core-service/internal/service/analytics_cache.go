package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/cache"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

const (
	defaultAnalyticsLiveTTL   = time.Minute
	defaultAnalyticsClosedTTL = 15 * time.Minute

	// defaultSalesSettleWindow bounds how far the report replica may trail the primary after facts change.
	defaultSalesSettleWindow = 30 * time.Second

	salesSettlingKeyPrefix = "core.analytics.sales_settling:"

	// analyticsLoadLeaseWait lets a burst of identical report requests across replicas run the aggregate once; a replica waits this long for another's result before computing its own.
	analyticsLoadLeaseWait = 15 * time.Second

	// analyticsAllScope is on every entry so a replica that may have missed events can drop the whole cache.
	analyticsAllScope = "analytics:all"
)

// analyticsFamily groups reports by the data they read, so a write invalidates only the reports it can change.
type analyticsFamily string

const (
	analyticsFamilySales      analyticsFamily = "sales"
	analyticsFamilyPricing    analyticsFamily = "pricing"
	analyticsFamilyProduction analyticsFamily = "production"
	analyticsFamilyDelivery   analyticsFamily = "delivery"
)

var analyticsFamilyResources = map[analyticsFamily][]constants.ObjectType{
	analyticsFamilySales: {
		constants.ObjectTypeInvoice, constants.ObjectTypeInvoiceLine, constants.ObjectTypeSalesOrder, constants.ObjectTypeSalesOrderLine,
		constants.ObjectTypeOrderDiscount, constants.ObjectTypeCustomer, constants.ObjectTypeAccountGroup, constants.ObjectTypeAccountUser,
		constants.ObjectTypeProduct, constants.ObjectTypeProductLine, constants.ObjectTypeItem, constants.ObjectTypeItemCategory,
		constants.ObjectTypeRate, constants.ObjectTypeQuantity, constants.ObjectTypeUnit, constants.ObjectTypeUnitGroup, constants.ObjectTypeAddress,
	},
	analyticsFamilyPricing: {
		constants.ObjectTypeAccountPrice, constants.ObjectTypeVolumeDiscount, constants.ObjectTypeCustomer, constants.ObjectTypeAccountGroup,
		constants.ObjectTypeProduct, constants.ObjectTypeProductLine, constants.ObjectTypeItem, constants.ObjectTypeRate,
		constants.ObjectTypeQuantity, constants.ObjectTypeUnit, constants.ObjectTypeUnitGroup,
	},
	analyticsFamilyProduction: {
		constants.ObjectTypeBatch, constants.ObjectTypeMachineDowntimeEvent, constants.ObjectTypeMachineDowntimeReason,
		constants.ObjectTypeProductionSchedule, constants.ObjectTypeProductionScheduleLine, constants.ObjectTypeProductionScheduleSettings,
		constants.ObjectTypeProductionScheduleDeviation, constants.ObjectTypeProductionStep, constants.ObjectTypeProductionRun,
		constants.ObjectTypeDepartment, constants.ObjectTypeMachine, constants.ObjectTypeScanningStation, constants.ObjectTypeItem,
		constants.ObjectTypeRate, constants.ObjectTypeQuantity, constants.ObjectTypeUnit, constants.ObjectTypeUnitGroup,
		constants.ObjectTypeItemCategory, constants.ObjectTypeConsumption, constants.ObjectTypeProduct, constants.ObjectTypeProductLine,
	},
	analyticsFamilyDelivery: {
		constants.ObjectTypeSalesOrder, constants.ObjectTypeSalesOrderLine, constants.ObjectTypePick, constants.ObjectTypePickLine,
		constants.ObjectTypeShipment, constants.ObjectTypeCustomer, constants.ObjectTypeAccountGroup, constants.ObjectTypeProduct,
		constants.ObjectTypeProductLine,
	},
}

var analyticsFamiliesByResource = func() map[constants.ObjectType][]analyticsFamily {
	out := make(map[constants.ObjectType][]analyticsFamily)
	for family, resources := range analyticsFamilyResources {
		for _, resource := range resources {
			out[resource] = append(out[resource], family)
		}
	}
	return out
}()

func analyticsScope(accountID string, family analyticsFamily) string {
	return "analytics:" + accountID + ":" + string(family)
}

// AnalyticsCacheConfig configures an AnalyticsCache.
type AnalyticsCacheConfig struct {
	// Store (optional; default: nil) holds the cached reports and should be shared by every replica. Nil disables caching.
	Store cache.Store

	// LiveTTL (optional; default: 1m) is the lifetime of a report whose window reaches into the last day or that reads the clock. Scans and dashboard writes change these without an audit event.
	LiveTTL time.Duration

	// ClosedTTL (optional; default: 15m) is the lifetime of a report whose window ended before yesterday; edits made without an audit event surface within it.
	ClosedTTL time.Duration

	// SalesSettleWindow (optional; default: 30s) is how long after an account's sales facts change its
	// sales reports are computed without being cached. Reports read a replica; one computed before the
	// replica caught up is served once rather than cached for its TTL.
	SalesSettleWindow time.Duration

	// Now (optional; default: time.Now) is the clock that decides whether a window is closed.
	Now func() time.Time
}

// WithDefaults returns a copy of the config with unset fields filled. It is safe to call on a nil receiver.
func (c *AnalyticsCacheConfig) WithDefaults() *AnalyticsCacheConfig {
	if c == nil {
		c = &AnalyticsCacheConfig{}
	}
	out := *c
	if out.LiveTTL == 0 {
		out.LiveTTL = defaultAnalyticsLiveTTL
	}
	if out.ClosedTTL == 0 {
		out.ClosedTTL = defaultAnalyticsClosedTTL
	}
	if out.SalesSettleWindow == 0 {
		out.SalesSettleWindow = defaultSalesSettleWindow
	}
	if out.Now == nil {
		out.Now = time.Now
	}
	return &out
}

func (c *AnalyticsCacheConfig) validate() error {
	if c.LiveTTL <= 0 || c.ClosedTTL <= 0 {
		return fmt.Errorf("analytics cache: TTLs must be positive")
	}
	return nil
}

// AnalyticsCache holds computed analytics reports per account. Reports are cached only after the caller's permissions are checked, and keyed on the effective parameters, so a sales rep's forced filters never serve another caller.
type AnalyticsCache struct {
	cfg AnalyticsCacheConfig

	sales          *cache.Cache[[]domain.SalesEntry]
	salesSummary   *cache.Cache[*domain.SalesSummary]
	salesBreakdown *cache.Cache[*domain.SalesBreakdown]
	newCustomers   *cache.Cache[*domain.NewCustomerPage]
	pricing        *cache.Cache[*domain.CustomerPricingAnalysis]
	oee            *cache.Cache[[]domain.OeeDepartment]
	oeeTrend       *cache.Cache[[]domain.OeeTrendPeriod]
	productionCost *cache.Cache[*domain.ProductionCostReport]
	attainment     *cache.Cache[*domain.ScheduleAttainmentResult]
	forecast       *cache.Cache[*domain.DemandForecastResult]
	weeksOfSales   *cache.Cache[*domain.WeeksOfSalesResult]
	delivery       *cache.Cache[*domain.DeliveryPerformanceResult]

	quarterlyOrders   *cache.Cache[[]domain.YearlyQuarterlyData]
	openOrdersSummary *cache.Cache[*domain.OpenOrdersSummary]
	openOrderProducts *cache.Cache[*domain.OpenOrderProductPage]
	openOrders        *cache.Cache[*domain.OpenOrderPage]
}

// NewAnalyticsCache returns an empty AnalyticsCache over cfg.Store.
func NewAnalyticsCache(cfg *AnalyticsCacheConfig) (*AnalyticsCache, error) {
	cfg = cfg.WithDefaults()
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	c := &AnalyticsCache{cfg: *cfg}
	var err error
	if c.sales, err = newAnalyticsReportCache[[]domain.SalesEntry](cfg, "sales"); err != nil {
		return nil, err
	}
	if c.salesSummary, err = newAnalyticsReportCache[*domain.SalesSummary](cfg, "sales_summary"); err != nil {
		return nil, err
	}
	if c.salesBreakdown, err = newAnalyticsReportCache[*domain.SalesBreakdown](cfg, "sales_breakdown"); err != nil {
		return nil, err
	}
	if c.newCustomers, err = newAnalyticsReportCache[*domain.NewCustomerPage](cfg, "new_customers"); err != nil {
		return nil, err
	}
	if c.pricing, err = newAnalyticsReportCache[*domain.CustomerPricingAnalysis](cfg, "customer_pricing"); err != nil {
		return nil, err
	}
	if c.oee, err = newAnalyticsReportCache[[]domain.OeeDepartment](cfg, "oee"); err != nil {
		return nil, err
	}
	if c.oeeTrend, err = newAnalyticsReportCache[[]domain.OeeTrendPeriod](cfg, "oee_trend"); err != nil {
		return nil, err
	}
	if c.productionCost, err = newAnalyticsReportCache[*domain.ProductionCostReport](cfg, "production_costs"); err != nil {
		return nil, err
	}
	if c.attainment, err = newAnalyticsReportCache[*domain.ScheduleAttainmentResult](cfg, "schedule_attainment"); err != nil {
		return nil, err
	}
	if c.forecast, err = newAnalyticsReportCache[*domain.DemandForecastResult](cfg, "demand_forecast"); err != nil {
		return nil, err
	}
	if c.weeksOfSales, err = newAnalyticsReportCache[*domain.WeeksOfSalesResult](cfg, "weeks_of_sales"); err != nil {
		return nil, err
	}
	if c.delivery, err = newAnalyticsReportCache[*domain.DeliveryPerformanceResult](cfg, "delivery_performance"); err != nil {
		return nil, err
	}
	if c.quarterlyOrders, err = newAnalyticsReportCache[[]domain.YearlyQuarterlyData](cfg, "quarterly_orders"); err != nil {
		return nil, err
	}
	if c.openOrdersSummary, err = newAnalyticsReportCache[*domain.OpenOrdersSummary](cfg, "open_orders_summary"); err != nil {
		return nil, err
	}
	if c.openOrderProducts, err = newAnalyticsReportCache[*domain.OpenOrderProductPage](cfg, "open_order_products"); err != nil {
		return nil, err
	}
	if c.openOrders, err = newAnalyticsReportCache[*domain.OpenOrderPage](cfg, "open_orders"); err != nil {
		return nil, err
	}
	return c, nil
}

func newAnalyticsReportCache[T any](cfg *AnalyticsCacheConfig, name string) (*cache.Cache[T], error) {
	return cache.New[T](&cache.Config{Name: "core.analytics." + name, Store: cfg.Store, TTL: cfg.LiveTTL, LoadLeaseWait: analyticsLoadLeaseWait})
}

// HandleAuditEvent drops the account's reports in every family the audited resource feeds.
func (c *AnalyticsCache) HandleAuditEvent(ctx context.Context, e audit.ObservedEvent) {
	if e.AccountID == "" {
		return
	}
	families := analyticsFamiliesByResource[e.ResourceType]
	if len(families) == 0 {
		return
	}
	scopes := make([]string, 0, len(families))
	for _, family := range families {
		scopes = append(scopes, analyticsScope(e.AccountID, family))
	}
	if err := cache.Invalidate(ctx, c.cfg.Store, scopes...); err != nil {
		slog.WarnContext(ctx, "analytics cache: invalidation failed; entries expire by TTL", "account_id", e.AccountID, "resource_type", e.ResourceType, "error", err)
	}
}

// InvalidateSales drops the accounts' cached sales reports. Called after sales_line_fact changes, since a report cached between an edit's audit event and the fact refresh was built from the old facts.
func (c *AnalyticsCache) InvalidateSales(ctx context.Context, accountIDs []string) {
	if c == nil {
		return
	}
	scopes := make([]string, len(accountIDs))
	for i, id := range accountIDs {
		scopes[i] = analyticsScope(id, analyticsFamilySales)
	}
	if err := cache.Invalidate(ctx, c.cfg.Store, scopes...); err != nil {
		slog.WarnContext(ctx, "analytics cache: sales invalidation failed; entries expire by TTL", "accounts", len(accountIDs), "error", err)
	}
	if c.cfg.Store == nil {
		return
	}
	for _, id := range accountIDs {
		if err := c.cfg.Store.Set(ctx, salesSettlingKeyPrefix+id, []byte{1}, c.cfg.SalesSettleWindow); err != nil {
			slog.WarnContext(ctx, "analytics cache: marking sales as settling failed", "account_id", id, "error", err)
		}
	}
}

// salesSettling reports whether the account's sales facts changed within the settle window, while the
// report replica may still be catching up. A failed lookup reads as settling: serving a report uncached
// costs a query, caching a stale one costs a TTL of wrong numbers.
func (c *AnalyticsCache) salesSettling(ctx context.Context, accountID string) bool {
	if c == nil || c.cfg.Store == nil {
		return false
	}
	_, ok, err := c.cfg.Store.Get(ctx, salesSettlingKeyPrefix+accountID)
	return ok || err != nil
}

// Flush drops every cached report, for when this replica may have missed audit events.
func (c *AnalyticsCache) Flush() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := cache.Invalidate(ctx, c.cfg.Store, analyticsAllScope); err != nil {
		slog.Warn("analytics cache: flush failed; entries expire by TTL", "error", err)
	}
}

// ttlForWindow trusts a report longer once its window can no longer gain new activity.
func (c *AnalyticsCache) ttlForWindow(windowEnd time.Time) time.Duration {
	now := c.cfg.Now().UTC()
	startOfYesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	if windowEnd.Before(startOfYesterday) {
		return c.cfg.ClosedTTL
	}
	return c.cfg.LiveTTL
}

type analyticsReport struct {
	accountID string
	family    analyticsFamily
	method    string
	params    any
	ttl       time.Duration
	// bypass computes the report without reading or writing the cache.
	bypass bool
}

// cachedReport runs load through rc, or runs it directly when caching is off or report.bypass is set. It must only be called after the caller's permission checks.
func cachedReport[T any](ctx context.Context, rc *cache.Cache[T], report analyticsReport, load func(context.Context) (T, *apierror.APIError)) (T, *apierror.APIError) {
	if report.bypass {
		return load(ctx)
	}
	id, err := analyticsReportID(report.method, report.params)
	if err != nil {
		return load(ctx)
	}
	key := cache.Key{
		Scopes: []string{analyticsAllScope, analyticsScope(report.accountID, report.family)},
		ID:     id,
		TTL:    report.ttl,
	}
	return rc.GetOrLoad(ctx, key, load)
}

func analyticsReportID(method string, params any) (string, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return method + ":" + hex.EncodeToString(sum[:16]), nil
}
