package sqs_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/open-mrp/api/shared/cloud/sqs"
)

// request is one decoded SQS API call: the operation from the X-Amz-Target header and the JSON body the SDK sent.
type request struct {
	Action string
	Body   map[string]any
}

// handler answers one call with an HTTP status and a JSON body.
type handler func(request) (int, string)

// fakeSQS stands in for the AWS endpoint. One server serves the whole package so
// AWS_ENDPOINT_URL_SQS can be set once, before any test runs, and the tests can
// still run in parallel: each claims its own queue and calls route by the
// QueueUrl the SDK puts in the request body.
type fakeSQS struct {
	mu       sync.Mutex
	handlers map[string]handler
}

var fake = &fakeSQS{handlers: map[string]handler{}}

func (f *fakeSQS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	queueURL, _ := body["QueueUrl"].(string)
	name := queueURL[strings.LastIndex(queueURL, "/")+1:]

	f.mu.Lock()
	h, ok := f.handlers[name]
	f.mu.Unlock()

	if !ok {
		http.Error(w, "unregistered queue "+name, http.StatusNotImplemented)
		return
	}

	_, action, _ := strings.Cut(r.Header.Get("X-Amz-Target"), ".")
	status, response := h(request{Action: action, Body: body})

	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, response)
}

func TestMain(m *testing.M) {
	server := httptest.NewServer(fake)

	os.Setenv("AWS_ACCESS_KEY_ID", "AKIAEXAMPLE")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	os.Setenv("AWS_REGION", "us-east-2")
	os.Setenv("AWS_ENDPOINT_URL_SQS", server.URL)

	code := m.Run()
	server.Close()
	os.Exit(code)
}

// newClient binds a client to a queue named after the calling test, so parallel
// tests never see each other's calls.
func newClient(t *testing.T, h handler) *sqs.Client {
	t.Helper()

	name := strings.ToLower(strings.NewReplacer("/", "-", "_", "-").Replace(t.Name()))

	fake.mu.Lock()
	fake.handlers[name] = h
	fake.mu.Unlock()

	t.Cleanup(func() {
		fake.mu.Lock()
		delete(fake.handlers, name)
		fake.mu.Unlock()
	})

	client, apiErr := sqs.NewClient(context.Background(), "us-east-2", "https://sqs.us-east-2.amazonaws.com/123456789012/"+name)
	if apiErr != nil {
		t.Fatalf("NewClient: %v", apiErr)
	}
	return client
}

// Every field of a received message is optional on the wire; a message missing
// its receipt handle must arrive as an empty string, not a panic.
func TestReceiveTranslatesMessages(t *testing.T) {
	t.Parallel()

	var got request
	client := newClient(t, func(r request) (int, string) {
		got = r
		return http.StatusOK, `{"Messages":[
			{"MessageId":"m1","Body":"first","ReceiptHandle":"rh1"},
			{"MessageId":"m2"}
		]}`
	})

	msgs, apiErr := client.Receive(context.Background(), 10, 20)
	if apiErr != nil {
		t.Fatalf("Receive: %v", apiErr)
	}

	if got.Action != "ReceiveMessage" {
		t.Errorf("action: got %q want ReceiveMessage", got.Action)
	}
	if got.Body["MaxNumberOfMessages"] != float64(10) || got.Body["WaitTimeSeconds"] != float64(20) {
		t.Errorf("poll parameters: got %v", got.Body)
	}
	if want := "https://sqs.us-east-2.amazonaws.com/123456789012/testreceivetranslatesmessages"; got.Body["QueueUrl"] != want {
		t.Errorf("QueueUrl: got %v want %q", got.Body["QueueUrl"], want)
	}

	if len(msgs) != 2 {
		t.Fatalf("messages: got %d want 2", len(msgs))
	}
	if msgs[0].Body != "first" || msgs[0].ReceiptHandle != "rh1" {
		t.Errorf("first message: got %+v", msgs[0])
	}
	if msgs[1].Body != "" || msgs[1].ReceiptHandle != "" {
		t.Errorf("message with no body or handle: got %+v", msgs[1])
	}
}

// An idle queue answers every long poll with nothing; the loop must see an empty
// batch rather than an error.
func TestReceiveEmptyBatch(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(request) (int, string) { return http.StatusOK, `{}` })

	msgs, apiErr := client.Receive(context.Background(), 10, 20)
	if apiErr != nil {
		t.Fatalf("Receive: %v", apiErr)
	}
	if len(msgs) != 0 {
		t.Fatalf("messages: got %d want 0", len(msgs))
	}
}

func TestReceiveSurfacesFailure(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(request) (int, string) {
		return http.StatusBadRequest, `{"__type":"com.amazonaws.sqs#QueueDoesNotExist","message":"gone"}`
	})

	if _, apiErr := client.Receive(context.Background(), 10, 20); apiErr == nil {
		t.Fatal("expected an error when the queue rejects the poll")
	}
}

// Shutdown cancels the context mid-poll: the wrapper must report the failure so
// the loop can exit instead of treating it as an empty batch.
func TestReceiveHonoursCancelledContext(t *testing.T) {
	t.Parallel()

	var calls int
	client := newClient(t, func(request) (int, string) {
		calls++
		return http.StatusOK, `{}`
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	msgs, apiErr := client.Receive(ctx, 10, 20)
	if apiErr == nil {
		t.Fatal("expected an error for a cancelled poll")
	}
	if msgs != nil {
		t.Errorf("messages: got %+v want nil", msgs)
	}
	if calls != 0 {
		t.Errorf("calls: got %d want 0", calls)
	}
}

func TestDeleteAcknowledgesByReceiptHandle(t *testing.T) {
	t.Parallel()

	var got request
	client := newClient(t, func(r request) (int, string) {
		got = r
		return http.StatusOK, `{}`
	})

	if apiErr := client.Delete(context.Background(), "rh1"); apiErr != nil {
		t.Fatalf("Delete: %v", apiErr)
	}

	if got.Action != "DeleteMessage" {
		t.Errorf("action: got %q want DeleteMessage", got.Action)
	}
	if got.Body["ReceiptHandle"] != "rh1" {
		t.Errorf("ReceiptHandle: got %v want rh1", got.Body["ReceiptHandle"])
	}
}

// A delete that silently failed would leave the message to be redelivered and
// reprocessed; the caller only learns of it from the returned error.
func TestDeleteSurfacesFailure(t *testing.T) {
	t.Parallel()

	client := newClient(t, func(request) (int, string) {
		return http.StatusBadRequest, `{"__type":"com.amazonaws.sqs#ReceiptHandleIsInvalid","message":"expired"}`
	})

	if apiErr := client.Delete(context.Background(), "rh-expired"); apiErr == nil {
		t.Fatal("expected an error when the receipt handle is rejected")
	}
}
