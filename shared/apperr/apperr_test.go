package apperr

import (
	"net/http"
	"testing"

	"github.com/open-mrp/apikit/apierror"
	"github.com/stretchr/testify/assert"
)

func TestNewResourceArchivedError(t *testing.T) {
	err := NewResourceArchivedError("payment_term_id", "The payment term is archived.")

	assert.Equal(t, CodeResourceArchived, err.Code)
	assert.Equal(t, http.StatusUnprocessableEntity, err.Status())
	assert.Equal(t, "payment_term_id", err.Param)
	assert.Equal(t, []apierror.FieldError{{Param: "payment_term_id", Code: CodeResourceArchived, Message: "The payment term is archived."}}, err.Errors)
	assert.Equal(t, DocsBaseURL+"resource_archived", err.Code.DocURL())
}

func TestResourceArchivedField_CollectsIntoOneValidationError(t *testing.T) {
	err := apierror.NewValidationError("2 fields are invalid.",
		ResourceArchivedField("carrier_id", "The carrier is archived."),
		ResourceArchivedField("payment_term_id", "The payment term is archived."))

	assert.Equal(t, apierror.CodeValidationFailed, err.Code)
	assert.Equal(t, "carrier_id", err.Param)
	assert.Len(t, err.Errors, 2)
	assert.Equal(t, CodeResourceArchived, err.Errors[1].Code)
}

func TestNewInvalidStateTransitionError(t *testing.T) {
	err := NewInvalidStateTransitionError("A canceled order can't be issued.")

	assert.Equal(t, CodeInvalidStateTransition, err.Code)
	assert.Equal(t, http.StatusConflict, err.Status())
	assert.Equal(t, apierror.TypeInvalidRequest, err.Type)
	assert.Empty(t, err.Param)
}
