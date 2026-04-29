package service

import (
	"context"
	"testing"

	"github.com/augno/api/services/core-service/internal/domain"
	repositorymock "github.com/augno/api/services/core-service/internal/domain/mock/repository"
	apierror "github.com/augno/api/shared/errors"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestValidateCostRateUnits(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		numID       string
		denID       string
		dims        map[string]string
		dimsErr     *apierror.APIError
		wantParam   string
		wantSuccess bool
	}{
		{
			name:        "valid: currency over discrete",
			numID:       "un_usd",
			denID:       "un_each",
			dims:        map[string]string{"un_usd": "currency", "un_each": "discrete"},
			wantSuccess: true,
		},
		{
			name:      "missing numerator id",
			numID:     "",
			denID:     "un_each",
			wantParam: "unit_cost.numerator_unit_id",
		},
		{
			name:      "missing denominator id",
			numID:     "un_usd",
			denID:     "",
			wantParam: "unit_cost.denominator_unit_id",
		},
		{
			name:      "non-currency numerator",
			numID:     "un_each",
			denID:     "un_each",
			dims:      map[string]string{"un_each": "discrete"},
			wantParam: "unit_cost.numerator_unit_id",
		},
		{
			name:      "currency denominator",
			numID:     "un_usd",
			denID:     "un_usd",
			dims:      map[string]string{"un_usd": "currency"},
			wantParam: "unit_cost.denominator_unit_id",
		},
		{
			name:      "unknown numerator",
			numID:     "un_missing",
			denID:     "un_each",
			dims:      map[string]string{"un_each": "discrete"},
			wantParam: "unit_cost.numerator_unit_id",
		},
		{
			name:      "unknown denominator",
			numID:     "un_usd",
			denID:     "un_missing",
			dims:      map[string]string{"un_usd": "currency"},
			wantParam: "unit_cost.denominator_unit_id",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := repositorymock.NewMockUnitRepo(ctrl)
			if tc.numID != "" && tc.denID != "" {
				repo.EXPECT().
					GetDimensionCodes(gomock.Any(), gomock.Any()).
					Return(tc.dims, tc.dimsErr).
					Times(1)
			}

			err := ValidateCostRateUnits(context.Background(), repo, tc.numID, tc.denID, "unit_cost")

			if tc.wantSuccess {
				require.Nil(t, err)
				return
			}
			require.NotNil(t, err)
			require.Equal(t, tc.wantParam, err.Param)
		})
	}
}

// Compile-time check: ensure UnitRepo mock satisfies the domain interface.
var _ domain.UnitRepo = (*repositorymock.MockUnitRepo)(nil)
