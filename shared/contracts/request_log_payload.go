package contracts

// RequestLogPayload is the part of a request log kept in object storage rather than on the
// request_log row: the fields that are large, unbounded, and read only when one log is opened.
// The api-gateway writes it; platform-service reads it back for the log's body includes.
type RequestLogPayload struct {
	QueryJSON        *string `json:"query_json,omitempty"`
	RequestBodyJSON  *string `json:"request_body_json,omitempty"`
	ResponseBodyJSON *string `json:"response_body_json,omitempty"`
	StackTrace       *string `json:"stack_trace,omitempty"`
}

// RequestLogPayloadKey is the object key for a request log's payload. It derives from the log id
// alone so a re-sent log overwrites the same object.
func RequestLogPayloadKey(requestLogID string) string {
	return "request-logs/" + requestLogID + ".json.gz"
}
