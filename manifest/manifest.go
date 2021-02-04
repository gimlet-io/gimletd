package manifest

const PushEvent = "push"
const TagEvent = "tag"
const PREvent = "pr"

type Manifest struct {
	App       string                 `yaml:"app"`
	Env       string                 `yaml:"env"`
	Namespace string                 `yaml:"namespace"`
	Deploy    *Deploy                 `yaml:"deploy"`
	Chart     Chart                  `yaml:"chart"`
	Values    map[string]interface{} `yaml:"values"`
}

type Chart struct {
	Repository string `yaml:"repository"`
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
}

type Deploy struct {
	Branch string `yaml:"branch"` //master| '^(master|hotfix\/.+)$'
	Event  string `yaml:"event"`  //push/tag/pr
}
