package apiresource

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmbeddedList(t *testing.T) {
	t.Parallel()
	nums := func(n int) []int {
		out := make([]int, n)
		for i := range out {
			out[i] = i + 1
		}
		return out
	}
	const path = "/v1/operations/work-orders/wkor_1/jobs"

	short := NewEmbeddedList(nums(3), false, path, nil)
	assert.Len(t, short.Data, 3)
	assert.Nil(t, short.PageInfo.NextPageURL, "a list that fits has no next page")

	long := NewEmbeddedList(nums(11), false, path, func(last int) string { return "c" + strconv.Itoa(last) })
	assert.Len(t, long.Data, EmbeddedListLimit)
	require.NotNil(t, long.PageInfo.NextPageURL)
	assert.Equal(t, path+"?cursor=c10", *long.PageInfo.NextPageURL, "the link continues after the last embedded item")

	told := NewEmbeddedList(nums(10), true, path, nil)
	require.NotNil(t, told.PageInfo.NextPageURL)
	assert.Equal(t, path, *told.PageInfo.NextPageURL, "without a cursor the link opens the list")

	const filtered = "/v1/operations/machines?work_center_ids%5B%5D=wkcc_1"
	withQuery := NewEmbeddedList(nums(11), false, filtered, func(int) string { return "a+b" })
	require.NotNil(t, withQuery.PageInfo.NextPageURL)
	assert.Equal(t, filtered+"&cursor=a%2Bb", *withQuery.PageInfo.NextPageURL, "a filtered path keeps its filter and adds the cursor")

	noCursor := NewEmbeddedList(nums(11), false, filtered, func(int) string { return "" })
	require.NotNil(t, noCursor.PageInfo.NextPageURL)
	assert.Equal(t, filtered, *noCursor.PageInfo.NextPageURL, "an unknown cursor opens the filtered list")

	empty := NewEmbeddedList[int](nil, false, path, nil)
	assert.NotNil(t, empty.Data)
	assert.Empty(t, empty.Data)
}
