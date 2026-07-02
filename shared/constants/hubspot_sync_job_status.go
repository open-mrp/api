package constants

// HubspotSyncJobStatus represents the lifecycle state of a HubSpot backfill sync job.
type HubspotSyncJobStatus string

const (
	// HubspotSyncJobStatusPreviewing indicates the read-only matching pass is running.
	HubspotSyncJobStatusPreviewing HubspotSyncJobStatus = "previewing"
	// HubspotSyncJobStatusReviewPending indicates the job is awaiting review resolution and execute confirmation.
	HubspotSyncJobStatusReviewPending HubspotSyncJobStatus = "review_pending"
	// HubspotSyncJobStatusExecuting indicates the write phase is running.
	HubspotSyncJobStatusExecuting HubspotSyncJobStatus = "executing"
	// HubspotSyncJobStatusCompleted indicates the job finished successfully.
	HubspotSyncJobStatusCompleted HubspotSyncJobStatus = "completed"
	// HubspotSyncJobStatusFailed indicates the job stopped on an error and can be re-run.
	HubspotSyncJobStatusFailed HubspotSyncJobStatus = "failed"
)

func (m HubspotSyncJobStatus) IsValid() bool {
	switch m {
	case HubspotSyncJobStatusPreviewing, HubspotSyncJobStatusReviewPending, HubspotSyncJobStatusExecuting, HubspotSyncJobStatusCompleted, HubspotSyncJobStatusFailed:
		return true
	default:
		return false
	}
}

func (m HubspotSyncJobStatus) EnumValues() []string {
	return []string{
		string(HubspotSyncJobStatusPreviewing),
		string(HubspotSyncJobStatusReviewPending),
		string(HubspotSyncJobStatusExecuting),
		string(HubspotSyncJobStatusCompleted),
		string(HubspotSyncJobStatusFailed),
	}
}
