package apierror

import "strings"

// pairs one row of a bulk request with the failure it produced; a failure of the request
// as a whole carries no index
type RowError struct {
	// Zero-based row of the request this failure names. Absent for a failure of the whole request.
	Index *int `json:"index,omitzero"`
	// What went wrong — the same error object a synchronous error response carries.
	Error ResponseError `json:"error" validate:"required"`
}

// returns a representative instance for OpenAPI documentation generation
func (r RowError) SchemaExample() any {
	return RowError{Index: new(2), Error: ResponseError{}.SchemaExample().(ResponseError)}
}

// records a row that failed, carrying the canonical client-facing error object
func NewRowError(index int, apiErr *APIError) RowError {
	return RowError{Index: &index, Error: apiErr.ToResponseError()}
}

// records a failure that belongs to no single row, such as one that sinks the whole batch
func NewBatchError(apiErr *APIError) RowError {
	return RowError{Error: apiErr.ToResponseError()}
}

// collects the failures found across a bulk request's rows, keeping each one whole
type RowErrors struct {
	entries []RowError
}

// records the failure one row produced
func (e *RowErrors) Add(index int, apiErr *APIError) {
	e.entries = append(e.entries, NewRowError(index, apiErr))
}

// records a row's field-level validation failure against a row-indexed param
func (e *RowErrors) AddValidation(index int, param, message string) {
	e.Add(index, NewValidationErrorWithParam(message, param))
}

// reports whether any row failed
func (e *RowErrors) Any() bool {
	return len(e.entries) > 0
}

// hands back the collected failures, the same shape a job records
func (e *RowErrors) Entries() []RowError {
	return e.entries
}

// renders the collected failures as the one error a synchronous response carries, or nil
// when there are none. Param holds a single field, so it names the first offending row.
func (e *RowErrors) Summary(entityPlural string) *APIError {
	if len(e.entries) == 0 {
		return nil
	}

	var firstParam string
	clauses := make([]string, 0, len(e.entries))
	for _, entry := range e.entries {
		var param string
		if entry.Error.Param != nil {
			param = *entry.Error.Param
		}
		if firstParam == "" {
			firstParam = param
		}
		if param == "" {
			clauses = append(clauses, entry.Error.Message)
			continue
		}
		clauses = append(clauses, param+": "+entry.Error.Message)
	}

	return NewValidationErrorWithParam(
		"Invalid "+entityPlural+" — "+strings.Join(clauses, "; ")+".", firstParam)
}
