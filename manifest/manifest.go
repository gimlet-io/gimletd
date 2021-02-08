package manifest

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	helmCLI "helm.sh/helm/v3/pkg/cli"
	"path/filepath"
	"strings"
	"text/template"
)

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

func HelmTemplate(manifestString string, vars map[string]string) (string, error) {
	tpl, err := template.New("").Funcs(sprig.TxtFuncMap()).Parse(string(manifestString))
	if err != nil {
		return "", err
	}

	var templated bytes.Buffer
	err = tpl.Execute(&templated, vars)
	if err != nil {
		return "", err
	}

	var m Manifest
	err = yaml.Unmarshal(templated.Bytes(), &m)
	if err != nil {
		return "", fmt.Errorf("cannot parse manifest")
	}

	actionConfig := new(action.Configuration)
	client := action.NewInstall(actionConfig)

	client.DryRun = true
	client.ReleaseName = m.App
	client.Replace = true
	client.ClientOnly = true
	client.APIVersions = []string{}
	client.IncludeCRDs = false
	client.ChartPathOptions.RepoURL = m.Chart.Repository
	client.ChartPathOptions.Version = m.Chart.Version
	client.Namespace = m.Namespace

	var settings = helmCLI.New()
	cp, err := client.ChartPathOptions.LocateChart(m.Chart.Name, settings)
	if err != nil {
		return "", err
	}

	chartRequested, err := loader.Load(cp)
	if err != nil {
		return "", err
	}

	rel, err := client.Run(chartRequested, m.Values)
	if err != nil {
		return "", err
	}

	return rel.Manifest, nil
}

func SplitHelmOutput(input map[string]string) map[string]string {
	if len(input) != 1 {
		return input
	}

	const separator = "---\n# Source: "

	files := map[string]string{}

	for _, content := range input {
		if !strings.Contains(content, separator) {
			return input
		}

		parts := strings.Split(content, separator)
		for _, p := range parts {
			p := strings.TrimSpace(p)
			if p == "" {
				continue
			}

			filePath := strings.Split(p, "\n")[0]
			fileName := filepath.Base(filePath)
			files[fileName] = strings.Join(strings.Split(p, "\n")[1:], "\n")
		}
	}

	return files
}
