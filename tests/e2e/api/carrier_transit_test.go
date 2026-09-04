//go:build e2e

package api_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Carrier transit: a promised date is a delivery date, so an order has to leave
// early enough for the carrier to cover the lane. ship_by_date is the promised
// date less transit in business days, and transit_days / transit_source are the
// audit trail for how that was worked out.
//
// Transit is warmed out of band, off the sales-order-created and
// sales-order-shipping-updated events, so order create stays fast. Nothing reads
// the lane cache directly, so these tests observe it the only way a client can:
// by issuing an order and looking at what got stamped.
//
// Scenarios are selected by the ship-to postal code, which the Shippo stub keys
// on (see internal/infrastructure/stub/shippo.go). Every code outside its table
// gets the ordinary three-service answer, which is why the pre-existing
// ship-by tests are unaffected: they use SeedCarrierID, which has no Shippo
// account and so never reaches the stub at all.

const (
	// Postal codes the stub gives special answers for. Kept in step with the
	// ZipStub* constants in the stub package.
	zipStubNoRates        = "99910"
	zipStubNoTransit      = "99911"
	zipStubError          = "99912"
	zipStubSameDay        = "99913"
	zipStubUnknownService = "99914"
	// Any code the stub has no special case for; it quotes the normal services.
	zipStubNormal = "43215"
)

// transitAddress creates a ship-to address with a chosen postal code, which is
// how a test selects a stub scenario.
func transitAddress(t *testing.T, postalCode string) string {
	t.Helper()
	status, body, err := apiClient.Post(addressesPath, map[string]any{
		"name":          uniqueName("e2e-transit-ship-to"),
		"street_line_1": "1 Transit Way",
		"locality":      "Columbus",
		"state":         "OH",
		"postal_code":   postalCode,
		"country":       "US",
	}, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, body)
	id := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(addressesPath + "/" + id) })
	return id
}

// createTransitOrder creates an estimate against the rateable carrier, shipping
// to the given postal code, with a promised delivery date. It does not issue:
// issuing is what stamps the commitment, and the lane has to warm first.
func createTransitOrder(t *testing.T, customerID, serviceLevelID, postalCode string, promisedAt time.Time) string {
	t.Helper()

	body := minimalSalesOrderCreateBody(t, customerID)
	body["carrier_id"] = SeedTransitCarrierID
	body["service_level_id"] = serviceLevelID
	body["ship_to_address_id"] = transitAddress(t, postalCode)
	body["promised_at"] = promisedAt.UTC().Format("2006-01-02") + "T00:00:00Z"

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "order create must not 5xx: %s", string(respBody))
	requireStatus(t, 201, status, respBody)

	orderID := jsonField(parseJSON(respBody), "id")
	deleteOrder(t, orderID)
	return orderID
}

func issueOrder(t *testing.T, orderID string) map[string]any {
	t.Helper()
	status, body, err := apiClient.Put(salesOrdersPath+"/"+orderID+"/actions/issue", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "issue must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
	return parseJSON(body)
}

func unissueOrder(t *testing.T, orderID string) {
	t.Helper()
	status, body, err := apiClient.Put(salesOrdersPath+"/"+orderID+"/actions/unissue", nil)
	require.NoError(t, err)
	require.Less(t, status, 500, "unissue must not 5xx: %s", string(body))
	requireStatus(t, 200, status, body)
}

// issueOnceWarm issues the order, retrying through unissue until the warmed lane
// shows up in the stamped commitment.
//
// A commitment is stamped at issue and never recomputed, so an order issued
// before its lane warmed keeps a transit-free date forever — correct behavior,
// and exactly what makes the cache unobservable by any other route. Withdrawing
// and re-issuing is the only way a client can ask again.
func issueOnceWarm(t *testing.T, orderID string) map[string]any {
	t.Helper()

	var issued map[string]any
	eventually(t, e2eAsyncWaitTimeout, e2eAsyncPollInterval, func() error {
		issued = issueOrder(t, orderID)
		if jsonField(commitmentOf(issued), "transit_days") == "" {
			unissueOrder(t, orderID)
			return fmt.Errorf("lane not warmed yet for order %s", orderID)
		}
		return nil
	})
	return issued
}

// awaitWarmBarrier proves the warm pipeline has drained past a point in time.
//
// Absence cannot be waited for, so a test that expects no transit needs to know
// the consumer has already had its chance. This creates a control order on a lane
// that definitely warms and blocks until it does. Because both orders publish to
// the same queue and the control is created second, the subject's warm was
// handled first — so once the control is warm, the subject has been decided.
func awaitWarmBarrier(t *testing.T, customerID string, promisedAt time.Time) {
	t.Helper()
	control := createTransitOrder(t, customerID, SeedTransitGroundServiceLevelID, zipStubNormal, promisedAt)
	issueOnceWarm(t, control)
}

// shipByFor is the promised date less n business days, which is what the
// commitment should come out to.
func shipByFor(promisedAt time.Time, n int) string {
	d := promisedAt.UTC().Truncate(24 * time.Hour)
	for range n {
		d = d.AddDate(0, 0, -1)
		for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			d = d.AddDate(0, 0, -1)
		}
	}
	return d.Format("2006-01-02")
}

// promisedMonday returns a Monday comfortably in the future, so every case
// crosses a weekend and business-day subtraction is actually exercised.
func promisedMonday() time.Time {
	d := time.Now().UTC().AddDate(0, 0, 40)
	for d.Weekday() != time.Monday {
		d = d.AddDate(0, 0, 1)
	}
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

// The whole point: the order leaves early enough to arrive when promised, and the
// weekend does not count as transit.
func TestCarrierTransit_ShipByIsPromisedDateLessTransit(t *testing.T) {
	t.Parallel()

	promised := promisedMonday()
	customerID := leadTimeCustomer(t, "e2e-transit-ground", ptrInt(30), "")
	orderID := createTransitOrder(t, customerID, SeedTransitGroundServiceLevelID, zipStubNormal, promised)

	issued := issueOnceWarm(t, orderID)

	assert.Equal(t, "3", jsonField(commitmentOf(issued), "transit_days"))
	assert.Equal(t, "carrier_lane", jsonField(commitmentOf(issued), "transit_source"))
	assert.Equal(t, "manual", jsonField(commitmentOf(issued), "lead_time_source"),
		"a promised date still beats the customer's standing rule")
	assert.Equal(t, shipByFor(promised, 3), shipByDate(t, issued))

	// The promised date is Monday and transit is three business days, so the
	// commitment lands on the previous Wednesday. Calendar arithmetic would have
	// said Friday, two days too late — the error this feature exists to remove.
	assert.NotEqual(t, promised.AddDate(0, 0, -3).Format("2006-01-02"), shipByDate(t, issued),
		"transit must be counted in business days, not calendar days")
}

// Each service on the lane gets its own transit from the one quote, so switching
// an order from ground to overnight moves the date it has to leave.
func TestCarrierTransit_FasterServiceShipsLater(t *testing.T) {
	t.Parallel()

	promised := promisedMonday()
	customerID := leadTimeCustomer(t, "e2e-transit-services", ptrInt(30), "")

	for _, tc := range []struct {
		name           string
		serviceLevelID string
		wantDays       string
		wantTransit    int
	}{
		{"ground", SeedTransitGroundServiceLevelID, "3", 3},
		{"two day", SeedTransitTwoDayServiceLevelID, "2", 2},
		{"overnight", SeedTransitOvernightServiceLevelID, "1", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			orderID := createTransitOrder(t, customerID, tc.serviceLevelID, zipStubNormal, promised)
			issued := issueOnceWarm(t, orderID)

			assert.Equal(t, tc.wantDays, jsonField(commitmentOf(issued), "transit_days"))
			assert.Equal(t, "carrier_lane", jsonField(commitmentOf(issued), "transit_source"))
			assert.Equal(t, shipByFor(promised, tc.wantTransit), shipByDate(t, issued))
		})
	}
}

// Zero is a real lane, not a missing one: it stamps a source, and the order ships
// the day it is due to arrive.
func TestCarrierTransit_SameDayLaneIsARealAnswer(t *testing.T) {
	t.Parallel()

	promised := promisedMonday()
	customerID := leadTimeCustomer(t, "e2e-transit-sameday", ptrInt(30), "")
	orderID := createTransitOrder(t, customerID, SeedTransitGroundServiceLevelID, zipStubSameDay, promised)

	issued := issueOnceWarm(t, orderID)

	assert.Equal(t, "0", jsonField(commitmentOf(issued), "transit_days"))
	assert.Equal(t, "carrier_lane", jsonField(commitmentOf(issued), "transit_source"))
	assert.Equal(t, promised.Format("2006-01-02"), shipByDate(t, issued))
}

// A service the carrier will not rate falls back to the number configured on the
// service level, which is the only transit such a lane will ever have.
func TestCarrierTransit_FallsBackToServiceLevelDefault(t *testing.T) {
	t.Parallel()

	promised := promisedMonday()
	customerID := leadTimeCustomer(t, "e2e-transit-default", ptrInt(30), "")
	orderID := createTransitOrder(t, customerID, SeedTransitDefaultOnlyServiceLevelID, zipStubNormal, promised)

	// The fallback needs no warm, so this stamps on the first issue.
	issued := issueOrder(t, orderID)

	assert.Equal(t, "5", jsonField(commitmentOf(issued), "transit_days"))
	assert.Equal(t, "service_level", jsonField(commitmentOf(issued), "transit_source"))
	assert.Equal(t, shipByFor(promised, 5), shipByDate(t, issued))
}

// With nothing known, the promised date is used as-is. That is how the system
// behaved before transit existed: visibly having no answer beats inventing one.
func TestCarrierTransit_UnknownTransitLeavesThePromisedDate(t *testing.T) {
	t.Parallel()

	promised := promisedMonday()
	customerID := leadTimeCustomer(t, "e2e-transit-unknown", ptrInt(30), "")
	orderID := createTransitOrder(t, customerID, SeedTransitNoTransitServiceLevelID, zipStubNormal, promised)

	awaitWarmBarrier(t, customerID, promised)
	issued := issueOrder(t, orderID)

	assert.Empty(t, jsonField(commitmentOf(issued), "transit_days"))
	assert.Empty(t, jsonField(commitmentOf(issued), "transit_source"))
	assert.Equal(t, promised.Format("2006-01-02"), shipByDate(t, issued),
		"with no transit the order is due to ship on the promised date itself")
}

// The carrier permutations that must all degrade to "no transit" rather than to a
// wrong date or a failed order.
func TestCarrierTransit_CarrierResponsesThatYieldNoEstimate(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		zip  string
		why  string
	}{
		{"carrier serves no rates for the lane", zipStubNoRates, "nothing to harvest"},
		{"carrier prices the lane but commits to no transit", zipStubNoTransit, "a price is not a delivery estimate"},
		{"carrier call fails", zipStubError, "a warm failure is swallowed, not surfaced"},
		{"carrier quotes a service the account does not carry", zipStubUnknownService, "the quote has nowhere to be filed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			promised := promisedMonday()
			customerID := leadTimeCustomer(t, "e2e-transit-neg", ptrInt(30), "")
			orderID := createTransitOrder(t, customerID, SeedTransitGroundServiceLevelID, tc.zip, promised)

			awaitWarmBarrier(t, customerID, promised)
			issued := issueOrder(t, orderID)

			assert.Empty(t, jsonField(commitmentOf(issued), "transit_days"), tc.why)
			assert.Empty(t, jsonField(commitmentOf(issued), "transit_source"), tc.why)
			assert.Equal(t, promised.Format("2006-01-02"), shipByDate(t, issued),
				"ship-by falls back to the promised date")
		})
	}
}

// A configured lead time is already a ship lead time, so subtracting transit from
// it would deduct the same journey twice and pull every defaulted order forward.
func TestCarrierTransit_LeadTimeOrdersAreUnaffected(t *testing.T) {
	t.Parallel()

	customerID := leadTimeCustomer(t, "e2e-transit-leadtime", ptrInt(14), "")

	// Warm the lane first, so the absence of transit below is a decision rather
	// than a race.
	awaitWarmBarrier(t, customerID, promisedMonday())

	body := minimalSalesOrderCreateBody(t, customerID)
	body["carrier_id"] = SeedTransitCarrierID
	body["service_level_id"] = SeedTransitGroundServiceLevelID
	body["ship_to_address_id"] = transitAddress(t, zipStubNormal)

	status, respBody, err := apiClient.Post(salesOrdersPath, body, newIdempotencyKey())
	require.NoError(t, err)
	requireStatus(t, 201, status, respBody)
	orderID := jsonField(parseJSON(respBody), "id")
	deleteOrder(t, orderID)

	issued := issueOrder(t, orderID)

	assert.Equal(t, "customer", jsonField(commitmentOf(issued), "lead_time_source"))
	assert.Equal(t, 14, committedRuleDays(t, issued))
	assert.Empty(t, jsonField(commitmentOf(issued), "transit_days"),
		"no delivery date was promised, so there is nothing to subtract transit from")
	assert.Equal(t, expectedShipBy(t, issued, 14), shipByDate(t, issued))
}

// Unissuing withdraws the whole commitment, transit included: an order off the
// book explains nothing about a date nobody is working to.
func TestCarrierTransit_UnissueClearsTransit(t *testing.T) {
	t.Parallel()

	promised := promisedMonday()
	customerID := leadTimeCustomer(t, "e2e-transit-unissue", ptrInt(30), "")
	orderID := createTransitOrder(t, customerID, SeedTransitGroundServiceLevelID, zipStubNormal, promised)

	issued := issueOnceWarm(t, orderID)
	require.Equal(t, "3", jsonField(commitmentOf(issued), "transit_days"), "precondition: transit was stamped")

	status, body, err := apiClient.Put(salesOrdersPath+"/"+orderID+"/actions/unissue", nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	after := parseJSON(body)
	assert.Empty(t, jsonField(commitmentOf(after), "transit_days"))
	assert.Empty(t, jsonField(commitmentOf(after), "transit_source"))
	assert.Empty(t, shipByDate(t, after))
}

// A commitment is a fact about the moment it was made. A carrier estimate that
// moves later must not drag a date the customer already has along with it.
func TestCarrierTransit_StampedCommitmentDoesNotMoveWithTheLane(t *testing.T) {
	t.Parallel()

	promised := promisedMonday()
	customerID := leadTimeCustomer(t, "e2e-transit-immutable", ptrInt(30), "")
	orderID := createTransitOrder(t, customerID, SeedTransitGroundServiceLevelID, zipStubNormal, promised)

	issued := issueOnceWarm(t, orderID)
	originalShipBy := shipByDate(t, issued)
	require.NotEmpty(t, originalShipBy)

	// Re-warming the same lane through another order must leave this one alone.
	awaitWarmBarrier(t, customerID, promised)

	status, body, err := apiClient.GetListRaw(salesOrdersPath+"/"+orderID, nil)
	require.NoError(t, err)
	requireStatus(t, 200, status, body)

	after := parseJSON(body)
	assert.Equal(t, originalShipBy, shipByDate(t, after))
	assert.Equal(t, "3", jsonField(commitmentOf(after), "transit_days"))
	assert.Equal(t, "carrier_lane", jsonField(commitmentOf(after), "transit_source"))
}

// The service level's fallback has to be settable, or the only transit an
// unratable lane can ever have is unreachable.
func TestCarrierTransit_ServiceLevelDefaultIsConfigurable(t *testing.T) {
	t.Parallel()

	created := createAndCleanup(t, carriersPath, map[string]any{
		"name": uniqueName("e2e-transit-carrier-cfg"),
		"code": "will_call",
	})
	carrierID := jsonField(created, "id")
	path := carriersPath + "/" + carrierID + "/service-levels"

	status, body, err := apiClient.Post(path, map[string]any{
		"name":                 uniqueName("e2e-transit-svc"),
		"code":                 uniqueName("e2e_transit_svc"),
		"default_transit_days": 4,
	}, newIdempotencyKey())
	require.NoError(t, err)
	require.Less(t, status, 500, "service level create must not 5xx: %s", string(body))
	requireStatus(t, 201, status, body)

	serviceLevelID := jsonField(parseJSON(body), "id")
	t.Cleanup(func() { _, _, _ = apiClient.Delete(path + "/" + serviceLevelID) })
	assert.Equal(t, "4", jsonField(parseJSON(body), "default_transit_days"))

	t.Run("updates", func(t *testing.T) {
		patchStatus, patchBody, err := apiClient.Patch(path+"/"+serviceLevelID,
			map[string]any{"default_transit_days": 6}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)
		assert.Equal(t, "6", jsonField(parseJSON(patchBody), "default_transit_days"))
	})

	// Omitting the field must not be read as clearing it, or any unrelated edit
	// would silently drop the only transit an unratable service has.
	t.Run("survives an unrelated edit", func(t *testing.T) {
		patchStatus, patchBody, err := apiClient.Patch(path+"/"+serviceLevelID,
			map[string]any{"name": uniqueName("e2e-transit-svc-renamed")}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)
		assert.Equal(t, "6", jsonField(parseJSON(patchBody), "default_transit_days"))
	})

	t.Run("clears explicitly", func(t *testing.T) {
		patchStatus, patchBody, err := apiClient.Patch(path+"/"+serviceLevelID,
			map[string]any{"default_transit_days": nil}, newIdempotencyKey())
		require.NoError(t, err)
		requireStatus(t, 200, patchStatus, patchBody)
		assert.Empty(t, jsonField(parseJSON(patchBody), "default_transit_days"))
	})

	t.Run("rejects a negative value", func(t *testing.T) {
		patchStatus, patchBody, err := apiClient.Patch(path+"/"+serviceLevelID,
			map[string]any{"default_transit_days": -1}, newIdempotencyKey())
		require.NoError(t, err)
		require.Less(t, patchStatus, 500, "a bad value must not 5xx: %s", string(patchBody))
		assert.Equal(t, 400, patchStatus, "body: %s", string(patchBody))
	})
}
