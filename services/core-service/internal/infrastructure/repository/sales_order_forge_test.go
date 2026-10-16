package repository

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/open-mrp/api/shared/constants"
)

func TestSalesOrderPaymentStatusPredicate(t *testing.T) {
	t.Parallel()

	assert.Empty(t, salesOrderPaymentStatusPredicate(nil), "no filter")
	all := []string{string(constants.SalesOrderPaymentStatusPaid), string(constants.SalesOrderPaymentStatusPartiallyPaid), string(constants.SalesOrderPaymentStatusUnpaid)}
	assert.Empty(t, salesOrderPaymentStatusPredicate(all), "every status is no filter")

	paid := salesOrderPaymentStatusPredicate([]string{string(constants.SalesOrderPaymentStatusPaid)})
	assert.Equal(t, "("+salesOrderPaidSQL+")", paid)

	unpaid := salesOrderPaymentStatusPredicate([]string{string(constants.SalesOrderPaymentStatusUnpaid)})
	assert.True(t, strings.HasPrefix(unpaid, "((NOT "+salesOrderPaidSQL+" AND NOT "), unpaid)

	both := salesOrderPaymentStatusPredicate([]string{string(constants.SalesOrderPaymentStatusPaid), string(constants.SalesOrderPaymentStatusPartiallyPaid)})
	assert.Contains(t, both, " OR ")
}

func TestLiveOrderAddress(t *testing.T) {
	t.Parallel()

	assert.Nil(t, liveOrderAddress("", new("HQ"), nil, nil, nil, nil, nil, new("US"), nil, nil), "an order naming no address has none")
	assert.Nil(t, liveOrderAddress("ad_1", new("HQ"), nil, nil, nil, nil, nil, nil, nil, nil), "an address that no longer joins has none")

	got := liveOrderAddress("ad_1", new("HQ"), new("1 Main"), nil, new("Columbus"), new("OH"), new("43204"), new("US"), nil, nil)
	assert.Equal(t, "ad_1", *got.AddressID)
	assert.Equal(t, "HQ", got.Name)
	assert.Equal(t, "Columbus", *got.City)
	assert.Equal(t, "US", got.Country)
}

func TestDocumentPartyIDIsStablePerParty(t *testing.T) {
	t.Parallel()

	a := documentPartyID("or_1", "ac_seller")
	assert.Equal(t, a, documentPartyID("or_1", "ac_seller"))
	assert.NotEqual(t, a, documentPartyID("or_1", "ac_buyer"))
	assert.True(t, strings.HasPrefix(a, "doac_"), a)
}
