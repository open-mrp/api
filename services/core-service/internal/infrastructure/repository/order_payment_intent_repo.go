package repository

import (
	"context"

	"github.com/augno/api/services/core-service/internal/domain"
	"github.com/augno/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var orderPaymentIntentRepoTracer = tracing.GetTracer("core-service.infrastructure.repository.order_payment_intent")

type orderPaymentIntentRepoImpl struct {
	queries *sqlc.Queries
}

func NewOrderPaymentIntentRepo(queries *sqlc.Queries) domain.OrderPaymentIntentRepo {
	return &orderPaymentIntentRepoImpl{queries: queries}
}

func (r *orderPaymentIntentRepoImpl) Create(ctx context.Context, id, paymentIntentID, salesOrderID string) *apierror.APIError {
	ctx, span := orderPaymentIntentRepoTracer.Start(ctx, "repository.order_payment_intent.create")
	defer span.End()

	err := r.queries.InsertOrderPaymentIntent(ctx, sqlc.InsertOrderPaymentIntentParams{
		ID:              id,
		PaymentIntentID: paymentIntentID,
		SalesOrderID:    salesOrderID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *orderPaymentIntentRepoImpl) FindByPaymentIntentID(ctx context.Context, paymentIntentID string) (*domain.OrderPaymentIntent, *apierror.APIError) {
	ctx, span := orderPaymentIntentRepoTracer.Start(ctx, "repository.order_payment_intent.find_by_payment_intent_id")
	defer span.End()

	row, err := r.queries.FindOrderPaymentIntentByPaymentIntentID(ctx, paymentIntentID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return &domain.OrderPaymentIntent{
		ID:              row.ID,
		PaymentIntentID: row.PaymentIntentID,
		SalesOrderID:    row.SalesOrderID,
	}, nil
}

func (r *orderPaymentIntentRepoImpl) Delete(ctx context.Context, id string) *apierror.APIError {
	ctx, span := orderPaymentIntentRepoTracer.Start(ctx, "repository.order_payment_intent.delete")
	defer span.End()

	err := r.queries.DeleteOrderPaymentIntent(ctx, id)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}
