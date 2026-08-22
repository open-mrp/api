package repository

import (
	"context"
	"database/sql"

	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/tracing"
)

var shippingCaseRepoTracer = tracing.GetTracer("core-service.shipping_case_repository")

type shippingCaseRepoImpl struct {
	queries *sqlc.Queries
}

func NewShippingCaseRepo(queries *sqlc.Queries) domain.ShippingCaseRepo {
	return &shippingCaseRepoImpl{queries: queries}
}

func (r *shippingCaseRepoImpl) Get(ctx context.Context, accountID, shippingCaseID string) (*domain.ShippingCase, *apierror.APIError) {
	ctx, span := shippingCaseRepoTracer.Start(ctx, "repository.shipping_case.get")
	defer span.End()

	row, err := r.queries.GetShippingCase(ctx, sqlc.GetShippingCaseParams{
		ID:        shippingCaseID,
		AccountID: accountID,
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	return mapGetShippingCaseRow(row), nil
}

func (r *shippingCaseRepoImpl) Update(ctx context.Context, params domain.UpdateShippingCaseParams) *apierror.APIError {
	ctx, span := shippingCaseRepoTracer.Start(ctx, "repository.shipping_case.update")
	defer span.End()

	var trackingNumber sql.NullString
	if params.TrackingNumber != nil {
		trackingNumber = sql.NullString{String: *params.TrackingNumber, Valid: true}
	}

	_, err := r.queries.UpdateShippingCaseTrackingNumber(ctx, sqlc.UpdateShippingCaseTrackingNumberParams{
		ID:             params.ShippingCaseID,
		AccountID:      params.AccountID,
		TrackingNumber: trackingNumber,
	})
	if err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}

	return nil
}

func (r *shippingCaseRepoImpl) RepointToCarrier(ctx context.Context, accountID, shipmentID, carrierID string) *apierror.APIError {
	ctx, span := shippingCaseRepoTracer.Start(ctx, "repository.shipping_case.repoint_to_carrier")
	defer span.End()

	err := r.queries.RepointShippingCasesToCarrier(ctx, sqlc.RepointShippingCasesToCarrierParams{
		CarrierID:  carrierID,
		ShipmentID: shipmentID,
		AccountID:  accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return nil
}

func (r *shippingCaseRepoImpl) Delete(ctx context.Context, accountID, shippingCaseID string) *apierror.APIError {
	ctx, span := shippingCaseRepoTracer.Start(ctx, "repository.shipping_case.delete")
	defer span.End()

	err := r.queries.DeleteShippingCase(ctx, sqlc.DeleteShippingCaseParams{
		ID:        shippingCaseID,
		AccountID: accountID,
	})
	if err != nil {
		return tracing.Trace(span, db.MapSQLError(err))
	}

	return nil
}

func (r *shippingCaseRepoImpl) IsInAccount(ctx context.Context, accountID, shippingCaseID string) (bool, *apierror.APIError) {
	ctx, span := shippingCaseRepoTracer.Start(ctx, "repository.shipping_case.is_in_account")
	defer span.End()

	exists, err := r.queries.CheckShippingCaseInAccount(ctx, sqlc.CheckShippingCaseInAccountParams{
		ID:        shippingCaseID,
		AccountID: accountID,
	})
	if err != nil {
		return false, tracing.Trace(span, db.MapSQLError(err))
	}

	return exists, nil
}

func (r *shippingCaseRepoImpl) GetNumber(ctx context.Context, accountID, shippingCaseID string) (string, *apierror.APIError) {
	ctx, span := shippingCaseRepoTracer.Start(ctx, "repository.shipping_case.get_number")
	defer span.End()

	number, err := r.queries.GetShippingCaseNumber(ctx, sqlc.GetShippingCaseNumberParams{
		ID:        shippingCaseID,
		AccountID: accountID,
	})
	if err != nil {
		return "", tracing.Trace(span, db.MapSQLError(err))
	}

	return number, nil
}

func (r *shippingCaseRepoImpl) ListByShipment(ctx context.Context, shipmentID string) ([]*domain.ShippingCase, *apierror.APIError) {
	ctx, span := shippingCaseRepoTracer.Start(ctx, "repository.shipping_case.list_by_shipment")
	defer span.End()

	rows, err := r.queries.ListShippingCasesByShipment(ctx, shipmentID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	cases := make([]*domain.ShippingCase, len(rows))
	for i, row := range rows {
		sc := &domain.ShippingCase{
			ID:                                 row.ID,
			Number:                             row.Number,
			ShipmentID:                         row.ShipmentID,
			CarrierID:                          row.CarrierID,
			CarrierName:                        row.CarrierName,
			CarrierIsPortalEnabled:             row.CarrierIsPortalEnabled,
			CarrierCreatedAt:                   row.CarrierCreatedAt,
			CarrierUpdatedAt:                   row.CarrierUpdatedAt,
			AccountID:                          row.AccountID,
			CreatedAt:                          row.CreatedAt,
			UpdatedAt:                          row.UpdatedAt,
			FreightAmountID:                    row.FreightAmountID,
			FreightAmountValue:                 row.FreightAmountValue,
			FreightAmountUnitID:                row.FreightAmountUnitID,
			FreightAmountUnitName:              row.FreightAmountUnitName,
			FreightAmountUnitAbbreviation:      row.FreightAmountUnitAbbreviation,
			FreightAmountUnitType:              row.FreightAmountUnitType,
			FreightAmountUnitRatioNumerator:    row.FreightAmountUnitRatioNumerator,
			FreightAmountUnitRatioDenominator:  row.FreightAmountUnitRatioDenominator,
			FreightAmountUnitOffsetNumerator:   row.FreightAmountUnitOffsetNumerator,
			FreightAmountUnitOffsetDenominator: row.FreightAmountUnitOffsetDenominator,
			FreightAmountUnitCreatedAt:         row.FreightAmountUnitCreatedAt,
			FreightAmountUnitUpdatedAt:         row.FreightAmountUnitUpdatedAt,
			FreightWeightID:                    row.FreightWeightID,
			FreightWeightValue:                 row.FreightWeightValue,
			FreightWeightUnitID:                row.FreightWeightUnitID,
			FreightWeightUnitName:              row.FreightWeightUnitName,
			FreightWeightUnitAbbreviation:      row.FreightWeightUnitAbbreviation,
			FreightWeightUnitType:              row.FreightWeightUnitType,
			FreightWeightUnitRatioNumerator:    row.FreightWeightUnitRatioNumerator,
			FreightWeightUnitRatioDenominator:  row.FreightWeightUnitRatioDenominator,
			FreightWeightUnitOffsetNumerator:   row.FreightWeightUnitOffsetNumerator,
			FreightWeightUnitOffsetDenominator: row.FreightWeightUnitOffsetDenominator,
			FreightWeightUnitCreatedAt:         row.FreightWeightUnitCreatedAt,
			FreightWeightUnitUpdatedAt:         row.FreightWeightUnitUpdatedAt,
		}
		if row.Sscc.Valid {
			sc.SSCC = &row.Sscc.String
		}
		if row.TrackingNumber.Valid {
			sc.TrackingNumber = &row.TrackingNumber.String
		}
		if row.ShippoTransactionID.Valid {
			sc.ShippoTransactionID = &row.ShippoTransactionID.String
		}
		if row.ShippingLabelUrl.Valid {
			sc.ShippingLabelURL = &row.ShippingLabelUrl.String
		}
		if row.ShippedAt.Valid {
			sc.ShippedAt = &row.ShippedAt.Time
		}
		cases[i] = sc
	}

	return cases, nil
}

func (r *shippingCaseRepoImpl) MarkShippedByShipment(ctx context.Context, shipmentID string) *apierror.APIError {
	ctx, span := shippingCaseRepoTracer.Start(ctx, "repository.shipping_case.mark_shipped_by_shipment")
	defer span.End()

	err := r.queries.MarkShippingCasesShippedByShipment(ctx, shipmentID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *shippingCaseRepoImpl) VoidByShipment(ctx context.Context, shipmentID string) *apierror.APIError {
	ctx, span := shippingCaseRepoTracer.Start(ctx, "repository.shipping_case.void_by_shipment")
	defer span.End()

	err := r.queries.VoidShippingCasesByShipment(ctx, shipmentID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *shippingCaseRepoImpl) UpdateWithShipmentInfo(ctx context.Context, shippingCaseID, trackingNumber, shippoTransactionID, shippingLabelURL string) *apierror.APIError {
	ctx, span := shippingCaseRepoTracer.Start(ctx, "repository.shipping_case.update_with_shipment_info")
	defer span.End()

	err := r.queries.UpdateShippingCaseWithShipmentInfo(ctx, sqlc.UpdateShippingCaseWithShipmentInfoParams{
		TrackingNumber:      sql.NullString{String: trackingNumber, Valid: trackingNumber != ""},
		ShippoTransactionID: sql.NullString{String: shippoTransactionID, Valid: shippoTransactionID != ""},
		ShippingLabelUrl:    sql.NullString{String: shippingLabelURL, Valid: shippingLabelURL != ""},
		ID:                  shippingCaseID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *shippingCaseRepoImpl) AddSscc(ctx context.Context, shippingCaseID, sscc string) *apierror.APIError {
	ctx, span := shippingCaseRepoTracer.Start(ctx, "repository.shipping_case.add_sscc")
	defer span.End()

	err := r.queries.AddSsccToShippingCase(ctx, sqlc.AddSsccToShippingCaseParams{
		Sscc: sql.NullString{String: sscc, Valid: sscc != ""},
		ID:   shippingCaseID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *shippingCaseRepoImpl) FindAndIncrementSsccCounter(ctx context.Context, accountID string) (int64, *apierror.APIError) {
	ctx, span := shippingCaseRepoTracer.Start(ctx, "repository.shipping_case.find_and_increment_sscc_counter")
	defer span.End()

	prop, err := r.queries.GetSysPropertyByTypeCode(ctx, sqlc.GetSysPropertyByTypeCodeParams{
		TypeCode:  string(constants.SysPropertyTypeCodeSsccCount),
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	currentValue := int64(prop.Value)

	_, err = r.queries.IncrementSysPropertyValue(ctx, sqlc.IncrementSysPropertyValueParams{
		ID:        prop.ID,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}

	return currentValue + 1, nil
}

func (r *shippingCaseRepoImpl) DeleteByShipment(ctx context.Context, shipmentID string) *apierror.APIError {
	ctx, span := shippingCaseRepoTracer.Start(ctx, "repository.shipping_case.delete_by_shipment")
	defer span.End()

	err := r.queries.DeleteShippingCasesByShipment(ctx, shipmentID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func mapGetShippingCaseRow(row sqlc.GetShippingCaseRow) *domain.ShippingCase {
	sc := &domain.ShippingCase{
		ID:                                 row.ID,
		Number:                             row.Number,
		ShipmentID:                         row.ShipmentID,
		ShipmentNumber:                     row.ShipmentNumber,
		ShipmentStatusCode:                 row.ShipmentStatusCode,
		ShipmentStatusName:                 row.ShipmentStatusName,
		ShipmentCreatedAt:                  row.ShipmentCreatedAt,
		ShipmentUpdatedAt:                  row.ShipmentUpdatedAt,
		CarrierID:                          row.CarrierID,
		CarrierName:                        row.CarrierName,
		CarrierIsPortalEnabled:             row.CarrierIsPortalEnabled,
		CarrierCreatedAt:                   row.CarrierCreatedAt,
		CarrierUpdatedAt:                   row.CarrierUpdatedAt,
		AccountID:                          row.AccountID,
		CreatedAt:                          row.CreatedAt,
		UpdatedAt:                          row.UpdatedAt,
		FreightAmountID:                    row.FreightAmountID,
		FreightAmountValue:                 row.FreightAmountValue,
		FreightAmountUnitID:                row.FreightAmountUnitID,
		FreightAmountUnitName:              row.FreightAmountUnitName,
		FreightAmountUnitAbbreviation:      row.FreightAmountUnitAbbreviation,
		FreightAmountUnitType:              row.FreightAmountUnitType,
		FreightAmountUnitRatioNumerator:    row.FreightAmountUnitRatioNumerator,
		FreightAmountUnitRatioDenominator:  row.FreightAmountUnitRatioDenominator,
		FreightAmountUnitOffsetNumerator:   row.FreightAmountUnitOffsetNumerator,
		FreightAmountUnitOffsetDenominator: row.FreightAmountUnitOffsetDenominator,
		FreightAmountUnitCreatedAt:         row.FreightAmountUnitCreatedAt,
		FreightAmountUnitUpdatedAt:         row.FreightAmountUnitUpdatedAt,
		FreightWeightID:                    row.FreightWeightID,
		FreightWeightValue:                 row.FreightWeightValue,
		FreightWeightUnitID:                row.FreightWeightUnitID,
		FreightWeightUnitName:              row.FreightWeightUnitName,
		FreightWeightUnitAbbreviation:      row.FreightWeightUnitAbbreviation,
		FreightWeightUnitType:              row.FreightWeightUnitType,
		FreightWeightUnitRatioNumerator:    row.FreightWeightUnitRatioNumerator,
		FreightWeightUnitRatioDenominator:  row.FreightWeightUnitRatioDenominator,
		FreightWeightUnitOffsetNumerator:   row.FreightWeightUnitOffsetNumerator,
		FreightWeightUnitOffsetDenominator: row.FreightWeightUnitOffsetDenominator,
		FreightWeightUnitCreatedAt:         row.FreightWeightUnitCreatedAt,
		FreightWeightUnitUpdatedAt:         row.FreightWeightUnitUpdatedAt,
	}

	if row.Sscc.Valid {
		sc.SSCC = &row.Sscc.String
	}
	if row.TrackingNumber.Valid {
		sc.TrackingNumber = &row.TrackingNumber.String
	}
	if row.ShippoTransactionID.Valid {
		sc.ShippoTransactionID = &row.ShippoTransactionID.String
	}
	if row.ShippingLabelUrl.Valid {
		sc.ShippingLabelURL = &row.ShippingLabelUrl.String
	}
	if row.ShippedAt.Valid {
		sc.ShippedAt = &row.ShippedAt.Time
	}

	return sc
}

func (r *shippingCaseRepoImpl) GetSalesOrderID(ctx context.Context, accountID, shippingCaseID string) (string, *apierror.APIError) {
	ctx, span := shippingCaseRepoTracer.Start(ctx, "repository.shipping_case.get_sales_order_id")
	defer span.End()

	orderID, err := r.queries.GetSalesOrderIDByShippingCase(ctx, sqlc.GetSalesOrderIDByShippingCaseParams{
		ShippingCaseID: shippingCaseID,
		AccountID:      accountID,
	})
	if err != nil {
		return "", tracing.Trace(span, db.MapSQLError(err))
	}
	return orderID, nil
}
