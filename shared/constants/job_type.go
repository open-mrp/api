package constants

// JobType represents the execution type of a message queued async job.
type JobType string

const (
	// JobTypeBulkCreate indicates the job is a bulk creation of an object.
	JobTypeBulkCreate JobType = "bulkcreate"
	// JobTypeBulkUpsert indicates the job is a bulk upsert of an object.
	JobTypeBulkUpsert JobType = "bulkupsert"
	// JobTypeExport indicates the job renders a resource as a downloadable file.
	JobTypeExport JobType = "export"
)

func (m JobType) IsValid() bool {
	switch m {
	case JobTypeBulkCreate, JobTypeBulkUpsert, JobTypeExport:
		return true
	default:
		return false
	}
}

func (m JobType) EnumValues() []string {
	return []string{
		string(JobTypeBulkCreate),
		string(JobTypeBulkUpsert),
		string(JobTypeExport),
	}
}
