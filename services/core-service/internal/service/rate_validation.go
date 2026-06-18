package service

import (
	"context"
	"fmt"

	"github.com/augno/api/services/core-service/internal/domain"
	apierror "github.com/augno/api/shared/errors"
)

const unitDimensionCodeCurrency = "currency"

// ValidateCostRateUnits ensures the unit IDs on each side of a cost-typed rate match the conventional shape of money/quantity:
//
//   - numerator must be a currency unit (the "$" in "$5/ea")
//   - denominator must NOT be a currency unit (the "ea" in "$5/ea")
//
// Both unit IDs are looked up in a single round trip via UnitRepo. The repo is taken as a parameter so callers can pass a transactional repo when the validation runs inside a transaction (consistent with how InsertRate is used).
//
// fieldName is the request-body field this validation belongs to (e.g., "unit_cost", "labor_rate") and is surfaced in the error so clients can pin the failure to the right input.
func ValidateCostRateUnits(ctx context.Context, repo domain.UnitRepo, numeratorUnitID, denominatorUnitID, fieldName string) *apierror.APIError {
	if numeratorUnitID == "" {
		return apierror.NewValidationErrorWithParam(
			fmt.Sprintf("%s.numerator_unit_id is required.", fieldName),
			fieldName+".numerator_unit_id",
		)
	}
	if denominatorUnitID == "" {
		return apierror.NewValidationErrorWithParam(
			fmt.Sprintf("%s.denominator_unit_id is required.", fieldName),
			fieldName+".denominator_unit_id",
		)
	}

	ids := []string{numeratorUnitID}
	if denominatorUnitID != numeratorUnitID {
		ids = append(ids, denominatorUnitID)
	}

	dims, apiErr := repo.GetDimensionCodes(ctx, ids)
	if apiErr != nil {
		return apiErr
	}

	numDim, ok := dims[numeratorUnitID]
	if !ok {
		return apierror.NewValidationErrorWithParam(
			fmt.Sprintf("%s.numerator_unit_id references a unit that does not exist.", fieldName),
			fieldName+".numerator_unit_id",
		)
	}
	denDim, ok := dims[denominatorUnitID]
	if !ok {
		return apierror.NewValidationErrorWithParam(
			fmt.Sprintf("%s.denominator_unit_id references a unit that does not exist.", fieldName),
			fieldName+".denominator_unit_id",
		)
	}

	if numDim != unitDimensionCodeCurrency {
		return apierror.NewValidationErrorWithParam(
			fmt.Sprintf("%s.numerator_unit_id must be a currency unit; got dimension %q.", fieldName, numDim),
			fieldName+".numerator_unit_id",
		)
	}
	if denDim == unitDimensionCodeCurrency {
		return apierror.NewValidationErrorWithParam(
			fmt.Sprintf("%s.denominator_unit_id must not be a currency unit.", fieldName),
			fieldName+".denominator_unit_id",
		)
	}

	return nil
}
