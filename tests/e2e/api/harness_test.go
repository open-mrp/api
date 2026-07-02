//go:build e2e

package api_test

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
)

const (
	defaultBaseURL     = "http://localhost:8082"
	healthPollTimeout  = 60 * time.Second
	healthPollInterval = 2 * time.Second
	// warmupTimeout bounds how long we wait for the notification fan-out pipeline to deliver its
	// first row on a cold stack before giving up and running the tests anyway.
	warmupTimeout = 60 * time.Second
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

	// Warm the notification fan-out pipeline before running tests. /healthz only reflects the API
	// gateway being up; it does not guarantee the notification-service RabbitMQ fan-out consumer has
	// connected and begun materializing rows. On a cold stack that consumer can take longer than a
	// single test's async budget (e2eAsyncWaitTimeout) to deliver its first notification, which would
	// otherwise flake whichever notification test happens to run first. Absorb that cold start here, once.
	log.Println("Warming notification pipeline...")
	if err := warmNotificationPipeline(accountID); err != nil {
		log.Printf("WARNING: notification pipeline warm-up did not complete: %v", err)
	} else {
		log.Println("Notification pipeline warm")
	}

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

// warmNotificationPipeline logs in as the seeded user, sends a notification to themselves, and
// blocks until it appears in their feed — proving the async fan-out pipeline is delivering rows.
func warmNotificationPipeline(accountID string) error {
	loginResp, err := apiClient.PostFull(loginPath, map[string]any{
		"identifier": seedUserEmail,
		"password":   seedUserPassword,
	}, newIdempotencyKey())
	if err != nil {
		return fmt.Errorf("warm-up login request failed: %w", err)
	}
	if loginResp.StatusCode != http.StatusOK {
		return fmt.Errorf("warm-up login failed (status %d): %s", loginResp.StatusCode, string(loginResp.Body))
	}
	token := cookieValue(loginResp.Header["Set-Cookie"], accessTokenCookie)
	if token == "" {
		return fmt.Errorf("warm-up login did not set the %s cookie", accessTokenCookie)
	}
	user := apiClient.WithBearerToken(token, accountID)

	title := fmt.Sprintf("e2e-warmup-%d", time.Now().UnixNano())
	sendResp, err := user.PostFull(notificationsPath, map[string]any{
		"category": "order.updated",
		"target":   map[string]any{"type": "account_user", "id": SeedAccountUserID},
		"title":    title,
	}, newIdempotencyKey())
	if err != nil {
		return fmt.Errorf("warm-up notification send failed: %w", err)
	}
	if sendResp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("warm-up notification send failed (status %d): %s", sendResp.StatusCode, string(sendResp.Body))
	}

	deadline := time.Now().Add(warmupTimeout)
	for time.Now().Before(deadline) {
		status, body, err := user.GetListRaw(notificationsPath, url.Values{"limit": {"100"}})
		if err == nil && status == http.StatusOK {
			var list struct {
				Data []json.RawMessage `json:"data"`
			}
			if json.Unmarshal(body, &list) == nil {
				for _, raw := range list.Data {
					var m map[string]any
					if json.Unmarshal(raw, &m) == nil {
						if s, _ := m["title"].(string); s == title {
							return nil
						}
					}
				}
			}
		}
		time.Sleep(healthPollInterval)
	}
	return fmt.Errorf("warm-up notification %q was not delivered within %s", title, warmupTimeout)
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
