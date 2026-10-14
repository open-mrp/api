package domain

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-mrp/api/shared/apperr"
)

type orderStatus string

func TestRequireTransition(t *testing.T) {
	t.Parallel()
	assert.Nil(t, RequireTransition("sales order", orderStatus("draft"), "issue", "draft"))

	err := RequireTransition("sales order", orderStatus("canceled"), "issue", "draft")
	require.NotNil(t, err)
	assert.Equal(t, apperr.CodeInvalidStateTransition, err.Code)
	assert.Equal(t, http.StatusConflict, err.Status())
	assert.Equal(t, "This sales order is `canceled`; `issue` needs it to be `draft`.", err.PublicMessage)

	err = RequireTransition("job", orderStatus("completed"), "cancel", "planned", "released", "in_progress")
	require.NotNil(t, err)
	assert.Equal(t, "This job is `completed`; `cancel` needs it to be `planned`, `released` or `in_progress`.", err.PublicMessage)
}
