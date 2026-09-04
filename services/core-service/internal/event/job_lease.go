package event

// jobInboxLeaseSeconds is the inbox lease for the batch-job consumers, which run for minutes rather than the seconds the default assumes.
//
// A lease that lapses while its handler is still working invites a redelivery to start the same job alongside it, and none of these jobs is a single transaction that a completion race could roll back. Overshooting only delays the retry of a job whose process was killed outright; undershooting duplicates the work.
const jobInboxLeaseSeconds = 1800
