package dx

// Release contains all metadata about a release event
type Release struct {
	App string `json:"app"`
	Env string `json:"env"`

	ArtifactID string   `json:"artifactId"`
	TriggeredBy string `json:"triggeredBy"`

	Version    *Version `json:"version"`

	GitopsRef   string `json:"gitopsRef"`
	GitopsRepo  string `json:"gitopsRepo"`
	Created     int64  `json:"created,omitempty"`
}
