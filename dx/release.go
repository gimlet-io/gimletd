package dx

// Release contains all metadata about a release event
type Release struct {
	App string `json:"app"`
	Env string `json:"env"`

	ArtifactID  string `json:"artifactId"`
	TriggeredBy string `json:"triggeredBy"`

	Version *Version `json:"version"`

	GitopsRef  string `json:"gitopsRef"`
	GitopsRepo string `json:"gitopsRepo"`
	Created    int64  `json:"created,omitempty"`

	RolledBack bool `json:"rolledBack,omitempty"`
}

// ReleaseRequest contains all metadata about the release intent
type ReleaseRequest struct {
	Env         string `json:"env"`
	App         string `json:"app"`
	ArtifactID  string `json:"artifactId"`
	TriggeredBy string `json:"triggeredBy"`
}

// RollbackRequest contains all metadata about the rollback intent
type RollbackRequest struct {
	Env         string `json:"env"`
	App         string `json:"app"`
	TargetSHA   string `json:"targetSHA"`
	TriggeredBy string `json:"triggeredBy"`
}

//ReleaseStatus holds the info of a release
type ReleaseStatus struct {
	GitopsSha string `json:"gitopsSha"`
	Applied   bool   `json:"applied"`
}
