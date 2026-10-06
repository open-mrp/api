package repository

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apierror "github.com/open-mrp/api/shared/errors"
)

type lastInsertID int64

func (r lastInsertID) LastInsertId() (int64, error) { return int64(r), nil }
func (lastInsertID) RowsAffected() (int64, error)   { return 1, nil }

// fakeCounter is a sys_property counter and the numbers its records already use.
type fakeCounter struct {
	value      int64
	exists     bool
	inUse      map[string]bool
	highestHit int
}

func (f *fakeCounter) counter() numberCounter {
	return numberCounter{
		allocate: func(context.Context, string) (sql.Result, error) {
			if !f.exists {
				f.exists, f.value = true, 1
				return lastInsertID(1), nil
			}
			f.value++
			return lastInsertID(f.value), nil
		},
		inUse: func(_ context.Context, number string) (bool, error) { return f.inUse[number], nil },
		highest: func(context.Context) (int64, error) {
			f.highestHit++
			var highest int64
			for number := range f.inUse {
				if n, err := strconv.ParseInt(number, 10, 64); err == nil && n > highest {
					highest = n
				}
			}
			return highest, nil
		},
		raise: func(_ context.Context, _ string, value int32) error {
			f.exists = true
			f.value = max(f.value, int64(value))
			return nil
		},
	}
}

func TestNumberCounter_AllocatesWithoutReadingTheHighestNumber(t *testing.T) {
	f := &fakeCounter{value: 41, exists: true, inUse: map[string]bool{"41": true}}

	got, apiErr := f.counter().next(context.Background(), "sprp_1")

	require.Nil(t, apiErr)
	assert.Equal(t, int64(42), got)
	assert.Zero(t, f.highestHit, "the usual allocation never scans the records it numbers")
}

func TestNumberCounter_ANewCounterStartsAfterTheNumbersInUse(t *testing.T) {
	f := &fakeCounter{inUse: map[string]bool{"7": true, "1005": true, "PR-FC-001": true}}

	got, apiErr := f.counter().next(context.Background(), "sprp_1")

	require.Nil(t, apiErr)
	assert.Equal(t, int64(1006), got)
}

func TestNumberCounter_ANewCounterForAnAccountWithNoNumbersStartsAtOne(t *testing.T) {
	f := &fakeCounter{inUse: map[string]bool{"PR-FC-001": true}}

	got, apiErr := f.counter().next(context.Background(), "sprp_1")

	require.Nil(t, apiErr)
	assert.Equal(t, int64(1), got)
}

// A number someone typed in, or a record imported, ahead of the counter is skipped, not handed out twice.
func TestNumberCounter_SkipsPastANumberAlreadyInUse(t *testing.T) {
	f := &fakeCounter{value: 9, exists: true, inUse: map[string]bool{"10": true, "11": true, "12": true}}

	got, apiErr := f.counter().next(context.Background(), "sprp_1")

	require.Nil(t, apiErr)
	assert.Equal(t, int64(13), got)
	assert.Equal(t, int64(13), f.value, "the counter carries on from the number it handed out")
}

func TestNumberCounter_GivesUpWhenEveryNumberItTriesIsTaken(t *testing.T) {
	f := &fakeCounter{value: 1, exists: true}
	c := f.counter()
	c.inUse = func(context.Context, string) (bool, error) { return true, nil }

	_, apiErr := c.next(context.Background(), "sprp_1")

	require.NotNil(t, apiErr)
	assert.Equal(t, http.StatusConflict, apierror.GetHTTPStatusCode(apiErr.Code))
}
