package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
)

// activitySessionGap is the idle time that ends a session. Thirty minutes is the usual analytics convention, and matches how long someone plausibly steps away mid-evaluation and comes back to the same task.
const activitySessionGap = 30 * time.Minute

// activityNoiseRoutes are requests the dashboard makes on its own — polling for notifications, loading the shell, finishing registration — that say nothing about what the registrant chose to do.
var activityNoiseRoutes = map[string]bool{
	"/v1/identity/me":                             true,
	"/v1/identity/me/tenancy":                     true,
	"/v1/messaging/notifications":                 true,
	"/v1/messaging/notifications/{id}":            true,
	"/v1/messaging/announcements":                 true,
	"/v1/messaging/conversations":                 true,
	"/v1/auth/registration-sessions/{session_id}": true,
	"/v1/identity/accounts/{id}/favicon":          true,
	"/v1/auth/api-keys/actions/fetch-doc-api-key": true,
}

// summarizeAccountActivity reduces a registrant's requests to the activity a reviewer and the drafter read. requests must be oldest first; truncated reports that the list hit its read limit.
func summarizeAccountActivity(requests []domain.AccountRequest, sandboxAccountID *string, truncated bool) domain.AccountActivity {
	activity := domain.AccountActivity{TotalRequests: len(requests), Truncated: truncated}
	if len(requests) == 0 {
		return activity
	}

	isSandbox := func(r domain.AccountRequest) bool {
		return sandboxAccountID != nil && r.TargetAccountID == *sandboxAccountID
	}

	created := map[string]int{}
	updated := map[string]int{}
	deleted := map[string]int{}
	viewed := map[string]bool{}
	type errorKey struct {
		method, resource string
		status           int
	}
	errs := map[errorKey]int{}
	days := map[string]bool{}

	var session *domain.AccountActivitySession
	var sessionEnd time.Time
	for _, r := range requests {
		days[r.OccurredAt.UTC().Format(time.DateOnly)] = true

		if session == nil || r.OccurredAt.Sub(sessionEnd) > activitySessionGap {
			if session != nil {
				activity.Sessions = append(activity.Sessions, *session)
			}
			session = &domain.AccountActivitySession{Start: r.OccurredAt.UTC(), Sandbox: true}
		}
		session.Requests++
		session.Sandbox = session.Sandbox && isSandbox(r)
		session.Minutes = int(r.OccurredAt.Sub(session.Start).Round(time.Minute) / time.Minute)
		sessionEnd = r.OccurredAt

		switch r.NormalizedRoute {
		case "/v1/auth/api-keys/actions/fetch-doc-api-key":
			activity.ReadAPIDocs = true
		case "/v1/billing/plans":
			activity.ViewedPlans = true
		}
		if r.Method == "POST" && r.NormalizedRoute == "/v1/auth/api-keys" && r.StatusCode < 300 {
			activity.CreatedAPIKey = true
		}
		if activityNoiseRoutes[r.NormalizedRoute] {
			continue
		}

		resource := activityResource(r.NormalizedRoute)
		switch {
		case r.StatusCode >= 400:
			errs[errorKey{r.Method, resource, r.StatusCode}]++
		case r.Method == "GET":
			viewed[resource] = true
		case r.Method == "POST" && !strings.Contains(r.NormalizedRoute, "/actions/"):
			created[resource]++
		case r.Method == "PATCH" || r.Method == "PUT":
			updated[resource]++
		case r.Method == "DELETE":
			deleted[resource]++
		}
	}
	activity.Sessions = append(activity.Sessions, *session)

	activity.DaysActive = len(days)
	activity.ReturnedAfterFirstDay = requests[len(requests)-1].OccurredAt.Sub(requests[0].OccurredAt) >= 24*time.Hour
	activity.Created = sortedCounts(created)
	activity.Updated = sortedCounts(updated)
	activity.Deleted = sortedCounts(deleted)
	for resource := range viewed {
		activity.Viewed = append(activity.Viewed, resource)
	}
	sort.Strings(activity.Viewed)
	for k, n := range errs {
		activity.Errors = append(activity.Errors, domain.AccountActivityError{Method: k.method, Resource: k.resource, StatusCode: k.status, Count: n})
	}
	sort.Slice(activity.Errors, func(i, j int) bool {
		a, b := activity.Errors[i], activity.Errors[j]
		if a.Resource != b.Resource {
			return a.Resource < b.Resource
		}
		if a.Method != b.Method {
			return a.Method < b.Method
		}
		return a.StatusCode < b.StatusCode
	})
	return activity
}

// activityResource names a route by the collection it addresses: "/v1/operations/locations/{id}" and "/v1/operations/locations" are both "operations/locations".
func activityResource(route string) string {
	parts := strings.Split(strings.Trim(route, "/"), "/")
	kept := parts[:0]
	for i, p := range parts {
		if i == 0 && strings.HasPrefix(p, "v") || strings.HasPrefix(p, "{") {
			continue
		}
		if p == "actions" {
			break
		}
		kept = append(kept, p)
	}
	return strings.Join(kept, "/")
}

func sortedCounts(m map[string]int) []domain.AccountActivityCount {
	counts := make([]domain.AccountActivityCount, 0, len(m))
	for resource, n := range m {
		counts = append(counts, domain.AccountActivityCount{Resource: resource, Count: n})
	}
	sort.Slice(counts, func(i, j int) bool {
		if counts[i].Count != counts[j].Count {
			return counts[i].Count > counts[j].Count
		}
		return counts[i].Resource < counts[j].Resource
	})
	if len(counts) == 0 {
		return nil
	}
	return counts
}

// activityTimeline renders activity as short lines for the reviewer. It is deterministic so the reviewer can check the drafter's reading against what actually happened.
func activityTimeline(a domain.AccountActivity) []string {
	if a.TotalRequests == 0 {
		return []string{"No activity after registering."}
	}

	var lines []string
	for i, s := range a.Sessions {
		where := "live account"
		if s.Sandbox {
			where = "sandbox"
		}
		lines = append(lines, fmt.Sprintf("Session %d: %s, %d min, %d requests (%s)", i+1, s.Start.Format("Jan 2 15:04 MST"), s.Minutes, s.Requests, where))
	}
	if a.ReturnedAfterFirstDay {
		lines = append(lines, fmt.Sprintf("Came back on a later day (%d days active)", a.DaysActive))
	}
	if len(a.Created) > 0 {
		lines = append(lines, "Created: "+formatCounts(a.Created))
	}
	if len(a.Updated) > 0 {
		lines = append(lines, "Updated: "+formatCounts(a.Updated))
	}
	if len(a.Deleted) > 0 {
		lines = append(lines, "Deleted: "+formatCounts(a.Deleted))
	}
	if len(a.Viewed) > 0 {
		lines = append(lines, "Looked at: "+strings.Join(a.Viewed, ", "))
	}
	if a.CreatedAPIKey {
		lines = append(lines, "Created an API key")
	}
	if a.ReadAPIDocs {
		lines = append(lines, "Opened the API docs")
	}
	if a.ViewedPlans {
		lines = append(lines, "Looked at plans")
	}
	for _, e := range a.Errors {
		lines = append(lines, fmt.Sprintf("Error: %s %s returned %d (%d×)", e.Method, e.Resource, e.StatusCode, e.Count))
	}
	if a.Truncated {
		lines = append(lines, fmt.Sprintf("Only the first %d requests were read", a.TotalRequests))
	}
	return lines
}

func formatCounts(counts []domain.AccountActivityCount) string {
	parts := make([]string, len(counts))
	for i, c := range counts {
		parts[i] = fmt.Sprintf("%s (%d)", c.Resource, c.Count)
	}
	return strings.Join(parts, ", ")
}
