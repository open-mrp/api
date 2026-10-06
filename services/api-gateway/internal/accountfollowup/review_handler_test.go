package accountfollowup

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/platform"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeClient struct {
	review   *pb.AccountFollowupReview
	getErr   *apierror.APIError
	writeErr *apierror.APIError
	approved *pb.ApproveAccountFollowupRequest
	skipped  bool
}

func (c *fakeClient) GetAccountFollowupReview(_ context.Context, _ *pb.GetAccountFollowupReviewRequest, _ ...grpc.CallOption) (*pb.AccountFollowupReview, error) {
	if c.getErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(c.getErr)
	}
	return c.review, nil
}

func (c *fakeClient) ApproveAccountFollowup(_ context.Context, req *pb.ApproveAccountFollowupRequest, _ ...grpc.CallOption) (*pb.AccountFollowupReview, error) {
	if c.writeErr != nil {
		return nil, contracts.ConvertAPIErrorToGRPC(c.writeErr)
	}
	c.approved = req
	return c.review, nil
}

func (c *fakeClient) SkipAccountFollowup(_ context.Context, _ *pb.SkipAccountFollowupRequest, _ ...grpc.CallOption) (*pb.AccountFollowupReview, error) {
	c.skipped = true
	return c.review, nil
}

func pendingReview() *pb.AccountFollowupReview {
	now := timestamppb.New(time.Date(2026, 10, 4, 9, 0, 0, 0, time.UTC))
	return &pb.AccountFollowupReview{
		Id: "acfu_1", Status: "pending_review",
		RegistrantName: "Sam Example", RegistrantEmail: "sam@customer-example.org",
		AccountName: "Example Co", AccountId: "ac_live", RegisteredAt: now,
		Engagement: "medium", InternalSummary: "Set up locations.", Timeline: []string{"Created: operations/locations (6)"},
		DraftSubject: "your openmrp setup", DraftBody: "Hi Sam,\n\nThanks for trying OpenMRP.", ReviewExpiresAt: now,
	}
}

func post(h http.Handler, form url.Values) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, Path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestReviewPageShowsDraftWithoutActing(t *testing.T) {
	t.Parallel()
	client := &fakeClient{review: pendingReview()}
	rec := httptest.NewRecorder()

	NewReviewHandler(client).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Path+"?token=tok", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "Sam Example")
	require.Contains(t, body, "Created: operations/locations (6)")
	require.Contains(t, body, `value="your openmrp setup"`)
	require.Contains(t, body, `name="token" value="tok"`)
	require.Nil(t, client.approved, "opening the link must not send")
	require.False(t, client.skipped)
	require.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
}

func TestReviewApproveRedirectsToResult(t *testing.T) {
	t.Parallel()
	client := &fakeClient{review: pendingReview()}

	rec := post(NewReviewHandler(client), url.Values{"token": {"tok"}, "action": {"approve"}, "subject": {"edited"}, "body": {"edited body"}})

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, Path+"?token=tok", rec.Header().Get("Location"))
	require.Equal(t, "edited", client.approved.Subject)
	require.Equal(t, "edited body", client.approved.Body)
}

func TestReviewSkip(t *testing.T) {
	t.Parallel()
	client := &fakeClient{review: pendingReview()}

	rec := post(NewReviewHandler(client), url.Values{"token": {"tok"}, "action": {"skip"}})

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.True(t, client.skipped)
	require.Nil(t, client.approved)
}

func TestReviewValidationErrorKeepsEdits(t *testing.T) {
	t.Parallel()
	client := &fakeClient{review: pendingReview(), writeErr: apierror.NewValidationErrorWithParam("Subject must be a single line of at most 200 characters.", "subject")}

	rec := post(NewReviewHandler(client), url.Values{"token": {"tok"}, "action": {"approve"}, "subject": {""}, "body": {"my edited body"}})

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Contains(t, rec.Body.String(), "Subject must be a single line")
	require.Contains(t, rec.Body.String(), "my edited body")
}

func TestReviewBadLinks(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		err    *apierror.APIError
		status int
	}{
		"unknown token": {apierror.NewResourceNotFoundError("This review link is invalid."), http.StatusNotFound},
		"expired":       {apierror.NewExpiredTokenError("This review link has expired."), http.StatusGone},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			NewReviewHandler(&fakeClient{getErr: tc.err}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Path+"?token=x", nil))
			require.Equal(t, tc.status, rec.Code)
			require.Contains(t, rec.Body.String(), tc.err.PublicMessage)
		})
	}
}

func TestReviewSentShowsWhatWentOut(t *testing.T) {
	t.Parallel()
	review := pendingReview()
	review.Status = "sent"
	review.FinalSubject = "final subject"
	review.FinalBody = "final body"
	review.ReviewedAt = review.RegisteredAt
	rec := httptest.NewRecorder()

	NewReviewHandler(&fakeClient{review: review}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Path+"?token=tok", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "final body")
	require.NotContains(t, rec.Body.String(), "<form", "a sent follow-up cannot be approved again")
}
