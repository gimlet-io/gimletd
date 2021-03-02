package dx

// Release contains all metadata about a release event
type Release struct {
	App        string `json:"app"`
	Env        string `json:"env"`
	ArtifactID string `json:"artifactId"`
	Created    int64  `json:"created,omitempty"`

	RepositoryName string `json:"repositoryName"`
	SHA            string `json:"sha"`
	Branch         string `json:"branch"`

	TriggeredBy string `json:"triggeredBy"`
	GitopsRef   string `json:"gitopsRef"`
	GitopsRepo  string `json:"gitopsRepo"`
}
