package fieldpb

import (
	"testing"

	pb "github.com/open-mrp/api/shared/proto/core"
	"github.com/open-mrp/apikit/field"
	"github.com/open-mrp/apikit/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDate_RoundTrips(t *testing.T) {
	t.Parallel()
	d := timeutil.NewDate(2028, 2, 29)

	got, err := DateFromProto(DateToProto(d))
	require.NoError(t, err)
	assert.Equal(t, d, *got)

	assert.Nil(t, DatePtrToProto(nil))
	nilDate, err := DateFromProto(nil)
	require.NoError(t, err)
	assert.Nil(t, nilDate)
}

func TestDateFromProto_RejectsDaysThatDoNotExist(t *testing.T) {
	t.Parallel()
	for _, p := range []*pb.Date{
		{Year: 2027, Month: 2, Day: 29},
		{Year: 2026, Month: 13, Day: 1},
		{Year: 2026, Month: 4, Day: 0},
		{},
	} {
		_, err := DateFromProto(p)
		assert.Error(t, err, "%v", p)
	}
}

func TestDateClearable_RoundTrips(t *testing.T) {
	t.Parallel()
	d := timeutil.NewDate(2026, 10, 14)

	for name, f := range map[string]field.Clearable[timeutil.Date]{
		"unset": field.Unset[timeutil.Date](),
		"clear": field.Clear[timeutil.Date](),
		"set":   field.Set(d),
	} {
		got, err := DateClearableFromProto(DateClearableToProto(f))
		require.NoError(t, err, name)
		assert.Equal(t, f, got, name)
	}

	assert.Nil(t, DateClearableToProto(field.Unset[timeutil.Date]()))
	got, err := DateClearableFromProto(&pb.DatePatch{})
	require.NoError(t, err)
	assert.True(t, got.IsClear(), "a patch with no value clears")

	_, err = DateClearableFromProto(&pb.DatePatch{Value: &pb.Date{Year: 2026, Month: 2, Day: 30}})
	assert.Error(t, err)
}
