package repository

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/safeconv"
)

// numberCounter hands out an account's next number in a series kept in sys_property: production runs,
// customers.
//
// The counter is authoritative. The usual allocation is one upsert, which holds the counter's row lock
// until the transaction ends so concurrent allocators queue, and one point lookup confirming no record
// already carries the number. Only a counter that was just created, or one found behind a number someone
// typed in or imported, reads the highest number in use and moves past it; that read is a plain one, so
// allocating never share-locks the records it numbers.
type numberCounter struct {
	allocate func(ctx context.Context, sysPropertyID string) (sql.Result, error)
	inUse    func(ctx context.Context, number string) (bool, error)
	highest  func(ctx context.Context) (int64, error)
	raise    func(ctx context.Context, sysPropertyID string, value int32) error
}

// maxNumberAllocations bounds the retries after a number turns out to be taken. Catching the counter up
// leaves it past every number in use, so the next allocation is free unless another writer took it.
const maxNumberAllocations = 3

// next reserves the next free number. sysPropertyID names the counter row should this create it.
func (c numberCounter) next(ctx context.Context, sysPropertyID string) (int64, *apierror.APIError) {
	for range maxNumberAllocations {
		result, err := c.allocate(ctx, sysPropertyID)
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return 0, apiErr
		}
		number, err := result.LastInsertId()
		if err != nil {
			return 0, apierror.NewInternalError(err, "Could not read the allocated number.")
		}

		// A counter that answers 1 was created by this allocation: start it after the numbers the
		// account already uses rather than at the beginning.
		if number == 1 {
			moved, apiErr := c.catchUp(ctx, sysPropertyID, number)
			if apiErr != nil {
				return 0, apiErr
			}
			if moved {
				continue
			}
		}

		taken, err := c.inUse(ctx, strconv.FormatInt(number, 10))
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return 0, apiErr
		}
		if !taken {
			return number, nil
		}
		if _, apiErr := c.catchUp(ctx, sysPropertyID, number); apiErr != nil {
			return 0, apiErr
		}
	}
	return 0, apierror.NewResourceConflictError("Could not reserve a free number. Try again.")
}

// catchUp moves the counter to the highest number in use when that is at or past number, and reports
// whether it moved.
func (c numberCounter) catchUp(ctx context.Context, sysPropertyID string, number int64) (bool, *apierror.APIError) {
	highest, err := c.highest(ctx)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return false, apiErr
	}
	if highest < number {
		return false, nil
	}
	if apiErr := db.MapSQLError(c.raise(ctx, sysPropertyID, safeconv.Int64ToInt32(highest))); apiErr != nil {
		return false, apiErr
	}
	return true, nil
}

func productionRunNumbers(q *sqlc.Queries, accountID string) numberCounter {
	return numberCounter{
		allocate: func(ctx context.Context, sysPropertyID string) (sql.Result, error) {
			return q.AllocateNextProductionRunNumber(ctx, sqlc.AllocateNextProductionRunNumberParams{ID: sysPropertyID, AccountID: accountID})
		},
		inUse: func(ctx context.Context, number string) (bool, error) {
			count, err := q.CountProductionRunsByNumber(ctx, sqlc.CountProductionRunsByNumberParams{AccountID: accountID, Number: number})
			return count > 0, err
		},
		highest: func(ctx context.Context) (int64, error) {
			return q.HighestNumericProductionRunNumber(ctx, accountID)
		},
		raise: func(ctx context.Context, sysPropertyID string, value int32) error {
			return q.RaiseProductionRunNumberCounter(ctx, sqlc.RaiseProductionRunNumberCounterParams{ID: sysPropertyID, AccountID: accountID, Value: value})
		},
	}
}

func customerNumbers(q *sqlc.Queries, ownerAccountID string) numberCounter {
	return numberCounter{
		allocate: func(ctx context.Context, sysPropertyID string) (sql.Result, error) {
			return q.AllocateNextCustomerNumber(ctx, sqlc.AllocateNextCustomerNumberParams{ID: sysPropertyID, AccountID: ownerAccountID})
		},
		inUse: func(ctx context.Context, number string) (bool, error) {
			return q.CustomerExistsByExternalNumber(ctx, sqlc.CustomerExistsByExternalNumberParams{OwnerAccountID: ownerAccountID, ExternalNumber: number})
		},
		highest: func(ctx context.Context) (int64, error) {
			return q.HighestNumericCustomerNumber(ctx, ownerAccountID)
		},
		raise: func(ctx context.Context, sysPropertyID string, value int32) error {
			return q.RaiseCustomerNumberCounter(ctx, sqlc.RaiseCustomerNumberCounterParams{ID: sysPropertyID, AccountID: ownerAccountID, Value: value})
		},
	}
}
