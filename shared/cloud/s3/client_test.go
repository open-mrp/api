package s3_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/open-mrp/api/shared/cloud/s3"
)

// fakeS3 stands in for the AWS endpoint. One server serves the whole package so
// S3_ENDPOINT_URL can be set once, before any test runs, and the tests can still
// run in parallel: each claims its own bucket and requests route by the bucket
// in the path (the client uses path-style addressing whenever the endpoint is
// overridden).
type fakeS3 struct {
	mu       sync.Mutex
	handlers map[string]http.HandlerFunc
}

var fake = &fakeS3{handlers: map[string]http.HandlerFunc{}}

func (f *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bucket, _, _ := strings.Cut(strings.TrimPrefix(r.URL.EscapedPath(), "/"), "/")

	f.mu.Lock()
	handler, ok := f.handlers[bucket]
	f.mu.Unlock()

	if !ok {
		http.Error(w, "unregistered bucket "+bucket, http.StatusNotImplemented)
		return
	}
	handler(w, r)
}

func TestMain(m *testing.M) {
	server := httptest.NewServer(fake)

	os.Setenv("AWS_ACCESS_KEY_ID", "AKIAEXAMPLE")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	os.Setenv("AWS_REGION", "us-east-2")
	os.Setenv("S3_ENDPOINT_URL", server.URL)

	code := m.Run()
	server.Close()
	os.Exit(code)
}

// bucketFor registers handler under a bucket name derived from the calling test,
// so parallel tests never see each other's requests.
func bucketFor(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()

	bucket := strings.ToLower(strings.NewReplacer("/", "-", "_", "-").Replace(t.Name()))

	fake.mu.Lock()
	fake.handlers[bucket] = handler
	fake.mu.Unlock()

	t.Cleanup(func() {
		fake.mu.Lock()
		delete(fake.handlers, bucket)
		fake.mu.Unlock()
	})
	return bucket
}

func newClient(t *testing.T) *s3.Client {
	t.Helper()

	client, apiErr := s3.NewClient(context.Background(), "us-east-2")
	if apiErr != nil {
		t.Fatalf("NewClient: %v", apiErr)
	}
	return client
}

// Copy is the attachment-promote step: the source object travels in a header, not
// the path, so the header is the only place a wrong source can be caught.
func TestCopyAddressesSourceAndDestination(t *testing.T) {
	t.Parallel()

	var gotSource, gotPath, gotSSE string
	bucket := bucketFor(t, func(w http.ResponseWriter, r *http.Request) {
		gotSource = r.Header.Get("x-amz-copy-source")
		gotPath = r.URL.EscapedPath()
		gotSSE = r.Header.Get("x-amz-server-side-encryption")
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><CopyObjectResult><ETag>"e"</ETag></CopyObjectResult>`))
	})

	apiErr := newClient(t).Copy(context.Background(), bucket, "staging/upload.pdf", "attachments/final.pdf")
	if apiErr != nil {
		t.Fatalf("Copy: %v", apiErr)
	}

	if want := bucket + "/staging/upload.pdf"; gotSource != want {
		t.Errorf("x-amz-copy-source: got %q want %q", gotSource, want)
	}
	if want := "/" + bucket + "/attachments/final.pdf"; gotPath != want {
		t.Errorf("destination path: got %q want %q", gotPath, want)
	}
	// The destination must be encrypted even though the source already is; S3 does not carry SSE across a copy.
	if gotSSE != "AES256" {
		t.Errorf("x-amz-server-side-encryption: got %q want AES256", gotSSE)
	}
}

func TestCopySurfacesFailure(t *testing.T) {
	t.Parallel()

	bucket := bucketFor(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchKey</Code></Error>`))
	})

	apiErr := newClient(t).Copy(context.Background(), bucket, "staging/gone.pdf", "attachments/final.pdf")
	if apiErr == nil {
		t.Fatal("expected an error when the source object is missing")
	}
}

// A failed existence check must not read as "absent": the caller turns absent
// into a user-facing "attachment was not found" and logs nothing.
func TestFileExistsMapsResponseStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		body    string
		want    bool
		wantErr bool
	}{
		{name: "present", status: http.StatusOK, want: true},
		{name: "absent", status: http.StatusNotFound, want: false},
		// S3 answers 403 rather than 404 when the caller may read an object but not list the bucket, so a denied HEAD on a key that does not exist still means "absent".
		{name: "access denied", status: http.StatusForbidden, body: `<Error><Code>AccessDenied</Code></Error>`, want: false},
		{name: "server error", status: http.StatusInternalServerError, body: `<Error><Code>InternalError</Code></Error>`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bucket := bucketFor(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprint(w, tc.body)
			})

			exists, apiErr := newClient(t).FileExists(context.Background(), bucket, "attachments/final.pdf")
			if (apiErr != nil) != tc.wantErr {
				t.Fatalf("error: got %v want error=%v", apiErr, tc.wantErr)
			}
			if exists != tc.want {
				t.Errorf("exists: got %v want %v", exists, tc.want)
			}
		})
	}
}
