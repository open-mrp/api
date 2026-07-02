package constants

// HubspotCompanyReviewStatus represents the resolution state of a HubSpot company-match review.
type HubspotCompanyReviewStatus string

const (
	// HubspotCompanyReviewStatusPending indicates the review is awaiting a human decision.
	HubspotCompanyReviewStatusPending HubspotCompanyReviewStatus = "pending"
	// HubspotCompanyReviewStatusResolved indicates the review was resolved (linked or marked create-new).
	HubspotCompanyReviewStatusResolved HubspotCompanyReviewStatus = "resolved"
	// HubspotCompanyReviewStatusSkipped indicates the customer was excluded from the sync.
	HubspotCompanyReviewStatusSkipped HubspotCompanyReviewStatus = "skipped"
)

func (m HubspotCompanyReviewStatus) IsValid() bool {
	switch m {
	case HubspotCompanyReviewStatusPending, HubspotCompanyReviewStatusResolved, HubspotCompanyReviewStatusSkipped:
		return true
	default:
		return false
	}
}

func (m HubspotCompanyReviewStatus) EnumValues() []string {
	return []string{
		string(HubspotCompanyReviewStatusPending),
		string(HubspotCompanyReviewStatusResolved),
		string(HubspotCompanyReviewStatusSkipped),
	}
}
