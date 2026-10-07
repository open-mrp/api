package service

import (
	"context"
	"fmt"
	"reflect"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	apierror "github.com/open-mrp/api/shared/errors"
)

// inlineAddressChoice is one address a request may name by ID or send inline, but not both.
type inlineAddressChoice struct {
	hasID       bool
	inline      *domain.InlineAddressParams
	idParam     string
	inlineParam string
}

// checkInlineAddressChoices refuses an address sent both by ID and inline.
func checkInlineAddressChoices(choices ...inlineAddressChoice) *apierror.APIError {
	for _, c := range choices {
		if c.hasID && c.inline != nil {
			return apierror.NewValidationErrorWithParam(fmt.Sprintf("Send either %s or %s, not both.", c.idParam, c.inlineParam), c.inlineParam)
		}
	}
	return nil
}

// checkInlineAddressPair refuses a bill-to and ship-to that edit the same stored address two different ways, since the second edit would silently overwrite the first.
func checkInlineAddressPair(bill, ship *domain.InlineAddressParams, shipParam string) *apierror.APIError {
	if bill == nil || ship == nil || bill.ID == nil || ship.ID == nil || *bill.ID != *ship.ID {
		return nil
	}
	if reflect.DeepEqual(*bill, *ship) {
		return nil
	}
	return apierror.NewValidationErrorWithParam("The ship-to address updates the same address as the bill-to address with different fields.", shipParam+".id")
}

// saveInlineAddresses saves a record's inline bill-to and ship-to addresses into accountID and returns their IDs, "" for a side sent without one. Identical inline addresses are saved once and shared, so one new address entered for both sides is not created twice.
func saveInlineAddresses(ctx context.Context, med domain.AddressMed, accountID string, bill, ship *domain.InlineAddressParams, billParam, shipParam string) (string, string, *apierror.APIError) {
	if apiErr := checkInlineAddressPair(bill, ship, shipParam); apiErr != nil {
		return "", "", apiErr
	}

	var billID, shipID string
	if bill != nil {
		saved, apiErr := med.Save(ctx, accountID, *bill, billParam)
		if apiErr != nil {
			return "", "", apiErr
		}
		billID = saved.ID
	}
	if ship != nil {
		if bill != nil && reflect.DeepEqual(*bill, *ship) {
			return billID, billID, nil
		}
		saved, apiErr := med.Save(ctx, accountID, *ship, shipParam)
		if apiErr != nil {
			return "", "", apiErr
		}
		shipID = saved.ID
	}
	return billID, shipID, nil
}

// previewInlineAddresses validates a record's inline bill-to and ship-to addresses against accountID without writing them, so a bad one is refused before the record's own work starts.
func previewInlineAddresses(ctx context.Context, med domain.AddressMed, accountID string, bill, ship *domain.InlineAddressParams, billParam, shipParam string) *apierror.APIError {
	if apiErr := checkInlineAddressPair(bill, ship, shipParam); apiErr != nil {
		return apiErr
	}
	for _, side := range []struct {
		input *domain.InlineAddressParams
		param string
	}{{bill, billParam}, {ship, shipParam}} {
		if side.input == nil {
			continue
		}
		if _, apiErr := med.Preview(ctx, accountID, *side.input, side.param); apiErr != nil {
			return apiErr
		}
	}
	return nil
}
