package artifact

import (
	"github.com/gimlet-io/gimlet-cli/manifest"
)

type Version struct {
	SHA            string `json:"sha,omitempty"`
	Branch         string `json:"branch,omitempty"`
	PR             bool   `json:"pr,omitempty"`
	SourceBranch   string `json:"sourceBranch,omitempty"`
	AuthorName     string `json:"authorName,omitempty"`
	AuthorEmail    string `json:"authorEmail,omitempty"`
	CommitterName  string `json:"committerName,omitempty"`
	CommitterEmail string `json:"committerEmail,omitempty"`
	Message        string `json:"message,omitempty"`
	RepositoryName string `json:"repositoryName,omitempty"`
	URL            string `json:"url,omitempty"`
}

// Artifact that contains all metadata that can be later used for releasing and auditing
type Artifact struct {
	ID string `json:"id,omitempty"`

	// The releasable version
	Version Version `json:"version,omitempty"`

	// Arbitrary environment variables from CI
	Context map[string]string `json:"context,omitempty"`

	// The complete set of Gimlet environments from the Gimlet environment files
	Environments []*manifest.Manifest `json:"environments,omitempty"`

	// CI job information, test results, Docker image information, etc
	Items []map[string]interface{} `json:"items,omitempty"`
}
