package mediator

import (
	"testing"

	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAddressName(t *testing.T) {
	t.Parallel()

	name, apiErr := NormalizeAddressName("  Warehouse  ")
	require.Nil(t, apiErr)
	assert.Equal(t, "Warehouse", name)

	_, apiErr = NormalizeAddressName("")
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorTypeInvalidRequest, apiErr.Type)

	_, apiErr = NormalizeAddressName("   ")
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorTypeInvalidRequest, apiErr.Type)
}

func TestNormalizeOptionalAddressName(t *testing.T) {
	t.Parallel()

	name, apiErr := NormalizeOptionalAddressName(nil)
	require.Nil(t, apiErr)
	assert.Nil(t, name)

	input := " HQ "
	name, apiErr = NormalizeOptionalAddressName(&input)
	require.Nil(t, apiErr)
	require.NotNil(t, name)
	assert.Equal(t, "HQ", *name)

	blank := ""
	_, apiErr = NormalizeOptionalAddressName(&blank)
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorTypeInvalidRequest, apiErr.Type)
}
