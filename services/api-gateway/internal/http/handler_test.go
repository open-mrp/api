package httptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/shared/contracts"
)

type testRequest struct {
	ID    int    `json:"id" query:"id" path:"id" validate:"required"`
	Name  string `json:"name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
}

type testResponse struct {
	Success bool `json:"success"`
}

func TestConvertToHTTPHandler_BindingErrors(t *testing.T) {
	be := apiendpoint.BoundEndpoint[*testRequest, *testResponse]{
		Handler: func(ctx context.Context, req *testRequest) (*testResponse, *contracts.APIError) {
			return &testResponse{Success: true}, nil
		},
		Spec: apiendpoint.APIEndpoint[*testRequest, *testResponse]{
			SuccessStatusCode: http.StatusOK,
		},
	}

	handler := ConvertToHTTPHandler(be)

	t.Run("Query Binding Error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?id=not-an-int", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		errObj := resp["error"].(map[string]any)

		if errObj["param"] != "id" {
			t.Errorf("expected param 'id', got %v", errObj["param"])
		}
	})

	t.Run("JSON Decode Unknown Field Error", func(t *testing.T) {
		body := `{"id": 1, "name": "Test", "email": "test@example.com", "unknown": "field"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		errObj := resp["error"].(map[string]any)

		if errObj["param"] != "unknown" {
			t.Errorf("expected param 'unknown', got %v", errObj["param"])
		}
	})

	t.Run("JSON Decode Type Error", func(t *testing.T) {
		body := `{"id": "not-an-int", "name": "Test", "email": "test@example.com"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		errObj := resp["error"].(map[string]any)

		if errObj["param"] != "id" {
			t.Errorf("expected param 'id', got %v", errObj["param"])
		}
	})

	t.Run("Validation Error Single Field", func(t *testing.T) {
		body := `{"id": 1, "name": "Test", "email": "invalid-email"}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		errObj := resp["error"].(map[string]any)

		if errObj["param"] != "email" {
			t.Errorf("expected param 'email', got %v", errObj["param"])
		}
	})

	t.Run("Validation Error Multiple Fields", func(t *testing.T) {
		body := `{"id": 1}` // Missing name and email
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}

		var resp map[string]any
		json.Unmarshal(w.Body.Bytes(), &resp)
		errObj := resp["error"].(map[string]any)

		// Should pick the first field that fails
		if errObj["param"] == nil || (errObj["param"] != "name" && errObj["param"] != "email") {
			t.Errorf("expected param 'name' or 'email', got %v", errObj["param"])
		}
	})
}
