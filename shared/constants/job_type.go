package constants

// JobType represents the execution type of a message queued async job.
type JobType string

const (
	// JobTypeBulkCreate indicates the job is a bulk creation of an object.
	JobTypeBulkCreate JobType = "bulk_create"
	// JobTypeBulkUpsert indicates the job is a bulk upsert of an object.
	JobTypeBulkUpsert JobType = "bulk_upsert"
	// JobTypeExport indicates the job renders a resource as a downloadable file.
	JobTypeExport JobType = "export"
	// Packs a pick into a new shipment with its lines and cases.
	JobTypePackPick JobType = "pack_pick"
)

func (m JobType) IsValid() bool {
	switch m {
	case JobTypeBulkCreate, JobTypeBulkUpsert, JobTypeExport, JobTypePackPick:
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
		string(JobTypePackPick),
	}
}
