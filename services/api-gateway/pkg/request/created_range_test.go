package apirequest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatedRange(t *testing.T) {
	t.Parallel()
	jan := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	feb := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)

	assert.Nil(t, CreatedRange{}.ToProto())
	assert.Nil(t, CreatedRange{}.Validate())

	r := CreatedRange{CreatedAfter: &jan, CreatedBefore: &feb}
	require.Nil(t, r.Validate())
	p := r.ToProto()
	assert.True(t, p.GetCreatedAfter().AsTime().Equal(jan))
	assert.True(t, p.GetCreatedBefore().AsTime().Equal(feb))

	onlyAfter := CreatedRange{CreatedAfter: &feb}.ToProto()
	assert.Nil(t, onlyAfter.CreatedBefore)

	bad := CreatedRange{CreatedAfter: &feb, CreatedBefore: &jan}.Validate()
	require.NotNil(t, bad)
	assert.Equal(t, "created_before", bad.Param)
}
