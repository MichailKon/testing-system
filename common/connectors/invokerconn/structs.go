package invokerconn

type JobType int

const (
	CompileJob JobType = iota
	TestJob
)

type Job struct {
	ID       string  `json:"ID"`
	SubmitID uint    `json:"SubmitID" binding:"required"`
	Type     JobType `json:"JobType" binding:"required"`
	Test     uint64  `json:"Test"`

	// TODO: Add job dependency
}

type StatusResponse struct {
	// TODO: add some information about invoker state
	MaxNewJobs   uint64   `json:"MaxNewJobs"`
	ActiveJobIDs []string `json:"ActiveJobIDs"`
}
