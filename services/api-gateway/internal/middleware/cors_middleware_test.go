package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCORSMiddlewareExposesRetryAfter(t *testing.T) {
	handler := CORSMiddleware()(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/catalog/products", nil)
	req.Header.Set("Origin", "https://openmrp.ai")
	rec := httptest.NewRecorder()
	handler(rec, req)

	exposed := strings.Split(rec.Header().Get("Access-Control-Expose-Headers"), ",")
	for i := range exposed {
		exposed[i] = strings.TrimSpace(exposed[i])
	}
	for _, want := range []string{"Retry-After", "Content-Disposition", "Request-Id"} {
		found := false
		for _, got := range exposed {
			if strings.EqualFold(got, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Access-Control-Expose-Headers %q does not include %s", rec.Header().Get("Access-Control-Expose-Headers"), want)
		}
	}
}
