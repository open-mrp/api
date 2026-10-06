// Package accountfollowup serves the page a reviewer uses to approve, edit, or skip a drafted account follow-up.
//
// It sits outside the API routers on purpose: it is an internal HTML page reached from an emailed link, not part of the public API, so it has no OpenAPI entry, no versioning, and no session. The token in the link is the only authorization, and platform-service stores only its hash.
package accountfollowup

import (
	"context"
	_ "embed"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/open-mrp/api/shared/contracts"
	apierror "github.com/open-mrp/api/shared/errors"
	pb "github.com/open-mrp/api/shared/proto/platform"
)

const (
	// Path is where the review page is mounted.
	Path = "/account-followups/review"

	// maxFormBytes bounds a submitted form: a subject and a body capped at 10,000 characters by platform-service.
	maxFormBytes = 64 << 10

	rpcTimeout = 10 * time.Second
)

//go:embed review.html
var reviewHTML string

var reviewTemplate = template.Must(template.New("review").Funcs(template.FuncMap{
	"date": func(ts interface{ AsTime() time.Time }) string {
		return ts.AsTime().UTC().Format("Jan 2, 2006 15:04 MST")
	},
}).Parse(reviewHTML))

type reviewPage struct {
	Token   string
	Review  *pb.AccountFollowupReview
	Subject string
	Body    string
	Error   string
	Message string
}

type reviewHandler struct {
	client pb.AccountFollowupServiceClient
}

// NewReviewHandler returns the handler for Path. GET renders the follow-up and never changes it, because mail scanners open links before people do; only a POST from the page approves or skips.
func NewReviewHandler(client pb.AccountFollowupServiceClient) http.Handler {
	return &reviewHandler{client: client}
}

func (h *reviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	// The token is in the URL: keep it out of referrers, caches, and indexes, and the page out of frames.
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("Cache-Control", "no-store")
	header.Set("X-Robots-Tag", "noindex, nofollow")
	header.Set("X-Frame-Options", "DENY")
	header.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")

	ctx, cancel := context.WithTimeout(r.Context(), rpcTimeout)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		token := r.URL.Query().Get("token")
		review, err := h.client.GetAccountFollowupReview(ctx, &pb.GetAccountFollowupReviewRequest{Token: token})
		if err != nil {
			h.renderError(w, r, err)
			return
		}
		h.render(w, http.StatusOK, reviewPage{Token: token, Review: review, Subject: review.DraftSubject, Body: review.DraftBody})

	case http.MethodPost:
		r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
		if err := r.ParseForm(); err != nil {
			h.render(w, http.StatusBadRequest, reviewPage{Message: "The form could not be read."})
			return
		}
		token := r.PostForm.Get("token")
		subject, body := r.PostForm.Get("subject"), r.PostForm.Get("body")

		var err error
		switch r.PostForm.Get("action") {
		case "approve":
			_, err = h.client.ApproveAccountFollowup(ctx, &pb.ApproveAccountFollowupRequest{Token: token, Subject: subject, Body: body})
		case "skip":
			_, err = h.client.SkipAccountFollowup(ctx, &pb.SkipAccountFollowupRequest{Token: token})
		default:
			h.render(w, http.StatusBadRequest, reviewPage{Message: "Unknown action."})
			return
		}
		if err != nil {
			apiErr := contracts.ConvertGRPCError(ctx, err, "platform-service")
			if apiErr.Code == apierror.ErrorCodeValidationFailed {
				// Keep the reviewer's edits on screen alongside the problem.
				review, getErr := h.client.GetAccountFollowupReview(ctx, &pb.GetAccountFollowupReviewRequest{Token: token})
				if getErr != nil {
					h.renderError(w, r, getErr)
					return
				}
				h.render(w, http.StatusUnprocessableEntity, reviewPage{Token: token, Review: review, Subject: subject, Body: body, Error: apiErr.PublicMessage})
				return
			}
			h.renderError(w, r, err)
			return
		}
		// Post/redirect/get, so a refresh shows the result instead of resubmitting.
		http.Redirect(w, r, Path+"?token="+url.QueryEscape(token), http.StatusSeeOther)

	default:
		header.Set("Allow", "GET, POST")
		h.render(w, http.StatusMethodNotAllowed, reviewPage{Message: "Method not allowed."})
	}
}

func (h *reviewHandler) renderError(w http.ResponseWriter, r *http.Request, err error) {
	apiErr := contracts.ConvertGRPCError(r.Context(), err, "platform-service")
	switch apiErr.Code {
	case apierror.ErrorCodeResourceNotFound:
		h.render(w, http.StatusNotFound, reviewPage{Message: apiErr.PublicMessage})
	case apierror.ErrorCodeExpiredToken:
		h.render(w, http.StatusGone, reviewPage{Message: apiErr.PublicMessage})
	default:
		slog.ErrorContext(r.Context(), "Account follow-up review failed", "error", apiErr)
		h.render(w, http.StatusInternalServerError, reviewPage{Message: "Something went wrong. Nothing was sent; try again in a moment."})
	}
}

func (h *reviewHandler) render(w http.ResponseWriter, status int, page reviewPage) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := reviewTemplate.Execute(w, page); err != nil {
		slog.Error("Account follow-up review: failed to render page", "error", err)
	}
}
