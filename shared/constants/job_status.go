package constants

// JobStatus represents the execution status of a message queued async job.
type JobStatus string

const (
	// JobStatusQueued indicates the job is queued but not yet started.
	JobStatusQueued JobStatus = "queued"
	// JobStatusRunning indicates the job is currently executing.
	JobStatusRunning JobStatus = "running"
	// JobStatusCompleted indicates the job completed successfully.
	JobStatusCompleted JobStatus = "completed"
	// JobStatusFailed indicates the job failed.
	JobStatusFailed JobStatus = "failed"
	// JobStatusCanceled indicates the job has been canceled.
	JobStatusCanceled JobStatus = "canceled"
)

func (m JobStatus) IsValid() bool {
	switch m {
	case JobStatusQueued, JobStatusRunning, JobStatusCompleted, JobStatusFailed, JobStatusCanceled:
		return true
	default:
		return false
	}
}

func (m JobStatus) EnumValues() []string {
	return []string{
		string(JobStatusQueued),
		string(JobStatusRunning),
		string(JobStatusCompleted),
		string(JobStatusFailed),
		string(JobStatusCanceled),
	}
}
