package service

import (
	"testing"

	apierror "github.com/augno/api/shared/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAddressName(t *testing.T) {
	t.Parallel()

	name, apiErr := normalizeAddressName("  Warehouse  ")
	require.Nil(t, apiErr)
	assert.Equal(t, "Warehouse", name)

	_, apiErr = normalizeAddressName("")
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorTypeInvalidRequest, apiErr.Type)

	_, apiErr = normalizeAddressName("   ")
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorTypeInvalidRequest, apiErr.Type)
}

func TestNormalizeOptionalAddressName(t *testing.T) {
	t.Parallel()

	name, apiErr := normalizeOptionalAddressName(nil)
	require.Nil(t, apiErr)
	assert.Nil(t, name)

	input := " HQ "
	name, apiErr = normalizeOptionalAddressName(&input)
	require.Nil(t, apiErr)
	require.NotNil(t, name)
	assert.Equal(t, "HQ", *name)

	blank := ""
	_, apiErr = normalizeOptionalAddressName(&blank)
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorTypeInvalidRequest, apiErr.Type)
}
