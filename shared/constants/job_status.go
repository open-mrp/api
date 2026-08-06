package constants

// JobStatus represents the execution status of a message queued async job.
type JobStatus string

const (
	// JobStatusCreated indicates the job is queued but not yet started.
	JobStatusCreated JobStatus = "created"
	// JobStatusStarted indicates the job is currently executing.
	JobStatusStarted JobStatus = "started"
	// JobStatusCompleted indicates the job completed successfully.
	JobStatusCompleted JobStatus = "completed"
	// JobStatusFailed indicates the job failed.
	JobStatusFailed JobStatus = "failed"
	// JobStatusCancelled indicates the job has been cancelled.
	JobStatusCancelled JobStatus = "cancelled"
)

func (m JobStatus) IsValid() bool {
	switch m {
	case JobStatusCreated, JobStatusStarted, JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		return true
	default:
		return false
	}
}

func (m JobStatus) EnumValues() []string {
	return []string{
		string(JobStatusCreated),
		string(JobStatusStarted),
		string(JobStatusCompleted),
		string(JobStatusFailed),
		string(JobStatusCancelled),
	}
}
