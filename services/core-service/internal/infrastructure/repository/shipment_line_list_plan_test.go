//go:build plans

package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/pagination"
)

// The shipment-line list is scoped to one shipment, so its scope key bounds every read by that
// shipment's lines: 176 at most in production, which the corpus's largest shipment matches.
func shipmentLinePlanDims() []planDim[domain.ListShipmentLinesParams] {
	str := func(s string) *string { return &s }
	return []planDim[domain.ListShipmentLinesParams]{
		{"shipment", []planValue[domain.ListShipmentLinesParams]{
			{"one-line", func(p *domain.ListShipmentLinesParams) { p.ShipmentID = planFulID("sh", 6) }},
		}},
		{"search", []planValue[domain.ListShipmentLinesParams]{
			{"every", func(p *domain.ListShipmentLinesParams) { p.Query = str("SKU-") }},
			{"one", func(p *domain.ListShipmentLinesParams) { p.Query = str("SKU-0118") }},
			{"none", func(p *domain.ListShipmentLinesParams) { p.Query = str("zzz") }},
		}},
	}
}

func shipmentLinePlanCases() []planCase[domain.ListShipmentLinesParams] {
	mid := planFulCreatedAt(planFulBigShipment).Add(40*time.Hour + planFulBigShipmentLines/2*time.Second)
	cursor := func(dir pagination.Direction) func(*domain.ListShipmentLinesParams) {
		return func(p *domain.ListShipmentLinesParams) {
			c := pagination.EncodeStringCursor(pagination.StringCursor{OccurredAt: mid, ID: "shl_planful_~", Direction: dir})
			p.Cursor = &c
		}
	}
	return planCases(
		domain.ListShipmentLinesParams{AccountID: planFulAccount, ShipmentID: planFulID("sh", planFulBigShipment), Limit: 25},
		qualifiedPlanDims(shipmentLinePlanDims()),
		[]planValue[domain.ListShipmentLinesParams]{
			{"first", func(*domain.ListShipmentLinesParams) {}},
			{"deep-next", cursor(pagination.DirectionForward)},
			{"deep-prev", cursor(pagination.DirectionBackward)},
		},
	)
}

// TestShipmentLineList_ReadsAboutAPage holds every filter combination ListShipmentLines accepts to
// reading about a page of lines (listPlanSuite).
func TestShipmentLineList_ReadsAboutAPage(t *testing.T) {
	ensureFulfillmentCorpus(t)
	listPlanSuite[domain.ListShipmentLinesParams]{
		table: "shipment_line", scopeColumn: "shipment_id",
		from: "FROM shipment_line sl", alias: "sl",
		statement: func(query string) bool { return strings.Contains(query, "FROM shipment_line sl") },
		cases:     shipmentLinePlanCases(),
		limit:     func(p domain.ListShipmentLinesParams) int32 { return p.Limit },
		list: func(ctx context.Context, q *sqlc.Queries, p domain.ListShipmentLinesParams) error {
			if _, apiErr := NewShipmentLineRepo(q).List(ctx, p); apiErr != nil {
				return apiErr
			}
			return nil
		},
	}.run(t)
}
