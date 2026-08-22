package domain

import (
	"context"
	"time"

	apierror "github.com/open-mrp/api/shared/errors"
)

// TransitLane identifies the journey a transit estimate describes: one service level between two postal codes. Postal codes are compared as stored, so callers normalize before building one.
type TransitLane struct {
	CarrierOptionID string
	OriginCountry   string
	OriginPostal    string
	DestCountry     string
	DestPostal      string
}

// IsComplete reports whether the lane has every part needed to look up or warm an estimate. An incomplete lane is the normal state for an order that has no carrier chosen yet, not an error.
func (l TransitLane) IsComplete() bool {
	return l.CarrierOptionID != "" &&
		l.OriginCountry != "" && l.OriginPostal != "" &&
		l.DestCountry != "" && l.DestPostal != ""
}

// CarrierTransitCandidates is what the database knows about a lane: the cached estimate if one has ever been warmed, and the service level's standing default. Which one wins is a policy decision made above the repository, because it depends on how stale the cached row is.
type CarrierTransitCandidates struct {
	// LaneDays is the cached carrier estimate for this exact lane, nil when it has never been warmed.
	LaneDays *int
	// LaneSourceCode says how the cached row was obtained, so a refresh knows whether it may overwrite it.
	LaneSourceCode string
	// LaneRefreshedAt is when the cached row was last written, used to judge staleness.
	LaneRefreshedAt *time.Time
	// ServiceLevelDefaultDays is the fallback configured on the service level, nil when none is set.
	ServiceLevelDefaultDays *int
}

// TransitWarmer fills the lane cache out of band, so the estimate is already there by the time an order is issued.
//
// It is an interface in domain because the consumers that drive it live in the event package, which must not depend on the service package that implements it.
type TransitWarmer interface {
	// WarmForOrder quotes an order's lane with its carrier and records the transit for every service level the carrier returns. It is best-effort by contract: an order with no carrier, no Shippo integration, or an unratable lane is a no-op, not an error.
	WarmForOrder(ctx context.Context, accountID, salesOrderID string) *apierror.APIError
}

// UpsertTransitEstimateParams writes a harvested estimate for a lane. Operator-entered rows are never overwritten, so SourceCode decides whether the write lands.
type UpsertTransitEstimateParams struct {
	ID          string
	AccountID   string
	Lane        TransitLane
	TransitDays int
	SourceCode  string
}
