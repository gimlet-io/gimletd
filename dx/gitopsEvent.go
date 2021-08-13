package dx

type Status int

const (
	Success Status = iota
	Failure
)

type GitopsEvent struct {
	Manifest    *Manifest
	Artifact    *Artifact
	TriggeredBy string

	Status     Status
	StatusDesc string

	GitopsRef  string
	GitopsRepo string
}
