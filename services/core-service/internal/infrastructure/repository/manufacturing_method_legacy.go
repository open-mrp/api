package repository

import (
	"context"
	gosql "database/sql"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/id"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
)

// adoptedMethodID is the method a step written outside methods becomes, derived from the step so a re-run adopts it
// once.
func adoptedMethodID(stepID string) string {
	return id.DerivedID(id.ManufacturingMethodIDPrefix, "production_step:"+stepID)
}

func (r *manufacturingMethodRepoImpl) SyncLegacyStep(ctx context.Context, stepID string) *apierror.APIError {
	out, err := r.queries.GetLegacyStepOutput(ctx, stepID)
	if err == gosql.ErrNoRows {
		return nil
	}
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	methodID := out.ManufacturingMethodID.String
	if !out.ManufacturingMethodID.Valid {
		methodID = adoptedMethodID(stepID)
		version, apiErr := r.NextVersion(ctx, out.ItemID)
		if apiErr != nil {
			return apiErr
		}
		if err := r.queries.InsertAdoptedManufacturingMethod(ctx, sqlc.InsertAdoptedManufacturingMethodParams{
			ID: methodID, AccountID: out.AccountID, ItemID: out.ItemID, Version: version,
			OutputQuantityValue: out.QuantityValue, OutputQuantityUnitID: out.QuantityUnitID,
			ActivatedAt: gosql.NullTime{Time: out.CreatedAt, Valid: true}, CreatedAt: out.CreatedAt,
		}); err != nil {
			return db.MapSQLError(err)
		}
		if err := r.queries.SetStepMethod(ctx, sqlc.SetStepMethodParams{
			ManufacturingMethodID: gosql.NullString{String: methodID, Valid: true}, StepID: stepID,
		}); err != nil {
			return db.MapSQLError(err)
		}
	} else {
		cur, err := r.queries.GetManufacturingMethodItemAndOutput(ctx, methodID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return apiErr
		}
		if cur.ItemID != out.ItemID || cur.OutputQuantityValue != out.QuantityValue || cur.OutputQuantityUnitID != out.QuantityUnitID {
			version := cur.Version
			if cur.ItemID != out.ItemID {
				var apiErr *apierror.APIError
				if version, apiErr = r.NextVersion(ctx, out.ItemID); apiErr != nil {
					return apiErr
				}
			}
			if err := r.queries.SyncAdoptedMethodOutput(ctx, sqlc.SyncAdoptedMethodOutputParams{
				ID: methodID, ItemID: out.ItemID, Version: version,
				OutputQuantityValue: out.QuantityValue, OutputQuantityUnitID: out.QuantityUnitID,
			}); err != nil {
				return db.MapSQLError(err)
			}
		}
	}
	if err := r.queries.AdoptStepConsumptions(ctx, sqlc.AdoptStepConsumptionsParams{
		ManufacturingMethodID: gosql.NullString{String: methodID, Valid: true}, StepID: gosql.NullString{String: stepID, Valid: true},
	}); err != nil {
		return db.MapSQLError(err)
	}
	machineIDs, err := r.queries.ListStepMachineIDs(ctx, gosql.NullString{String: stepID, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	return r.SetOperationMachines(ctx, out.AccountID, stepID, machineIDs)
}

func (r *manufacturingMethodRepoImpl) DeleteIfEmpty(ctx context.Context, accountID, methodID string) *apierror.APIError {
	n, err := r.queries.CountMethodOperations(ctx, gosql.NullString{String: methodID, Valid: true})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return apiErr
	}
	if n > 0 {
		return nil
	}
	return r.Delete(ctx, accountID, methodID)
}

func (r *manufacturingMethodRepoImpl) ListStepIDsAfter(ctx context.Context, afterID string, limit int32) ([]string, *apierror.APIError) {
	ids, err := r.queries.ListUnadoptedStepIDsAfter(ctx, sqlc.ListUnadoptedStepIDsAfterParams{AfterID: afterID, Limit: limit})
	return ids, db.MapSQLError(err)
}
