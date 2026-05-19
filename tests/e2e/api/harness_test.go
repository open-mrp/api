//go:build e2e

package api_test

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	defaultBaseURL     = "http://localhost:8082"
	healthPollTimeout  = 60 * time.Second
	healthPollInterval = 2 * time.Second
)

var (
	apiClient       *Client
	listEndpoints   []ListEndpointSpec
	updateEndpoints []UpdateEndpointSpec
	createEndpoints []BodyEndpointSpec
	putEndpoints    []BodyEndpointSpec
)

func TestMain(m *testing.M) {
	baseURL := envOr("E2E_BASE_URL", defaultBaseURL)
	apiKey := envOr("E2E_API_KEY", SeedAPIKey)
	accountID := envOr("E2E_ACCOUNT_ID", SeedAccountID)

	apiClient = NewClient(baseURL, apiKey, accountID)

	// Wait for the API gateway to be healthy.
	log.Println("Waiting for API gateway to be healthy...")
	if err := waitForHealth(baseURL); err != nil {
		log.Fatalf("API gateway did not become healthy: %v", err)
	}
	log.Println("API gateway is healthy")

	// Verify authentication works before running tests.
	log.Println("Verifying authentication...")
	status, body, err := apiClient.GetListRaw("/v1/auth/api-keys", nil)
	if err != nil {
		log.Fatalf("Auth verification request failed: %v", err)
	}
	if status != 200 {
		log.Fatalf("Auth verification failed (status %d): %s", status, string(body))
	}
	log.Println("Authentication verified")

	// Load list endpoints from the OpenAPI spec.
	listEndpoints, err = LoadListEndpoints()
	if err != nil {
		log.Fatalf("Failed to load OpenAPI spec: %v", err)
	}
	if len(listEndpoints) == 0 {
		log.Fatal("No list endpoints found in OpenAPI spec")
	}
	log.Printf("Discovered %d list endpoints from OpenAPI spec", len(listEndpoints))

	// Load update endpoints from the OpenAPI spec.
	updateEndpoints, err = LoadUpdateEndpoints()
	if err != nil {
		log.Fatalf("Failed to load update endpoints: %v", err)
	}
	log.Printf("Discovered %d update endpoints from OpenAPI spec", len(updateEndpoints))

	createEndpoints, err = LoadBodyEndpoints("post", excludedCreateOperations)
	if err != nil {
		log.Fatalf("Failed to load create endpoints: %v", err)
	}
	log.Printf("Discovered %d create endpoints from OpenAPI spec", len(createEndpoints))

	putEndpoints, err = LoadBodyEndpoints("put", excludedPutOperations)
	if err != nil {
		log.Fatalf("Failed to load put endpoints: %v", err)
	}
	log.Printf("Discovered %d put endpoints from OpenAPI spec", len(putEndpoints))

	os.Exit(m.Run())
}

func waitForHealth(baseURL string) error {
	deadline := time.Now().Add(healthPollTimeout)
	healthURL := baseURL + "/healthz"

	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(healthPollInterval)
	}

	return fmt.Errorf("timeout after %s", healthPollTimeout)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
