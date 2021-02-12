package dx

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	giturl "github.com/whilp/git-urls"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	helmCLI "helm.sh/helm/v3/pkg/cli"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
	"text/template"
)

type Manifest struct {
	App       string                 `yaml:"app" json:"app"`
	Env       string                 `yaml:"env" json:"env"`
	Namespace string                 `yaml:"namespace" json:"namespace"`
	Deploy    *Deploy                `yaml:"deploy,omitempty" json:"deploy,omitempty"`
	Chart     Chart                  `yaml:"chart" json:"chart"`
	Values    map[string]interface{} `yaml:"values" json:"values"`
}

type Chart struct {
	Repository string `yaml:"repository" json:"repository"`
	Name       string `yaml:"name" json:"name"`
	Version    string `yaml:"version" json:"version"`
}

type Deploy struct {
	Branch string    `yaml:"branch,omitempty" json:"branch,omitempty"` //master| '^(master|hotfix\/.+)$'
	Event  *GitEvent `yaml:"event,omitempty" json:"event,omitempty"`
}

func (m *Manifest) ResolveVars(vars map[string]string) error {
	manifestString, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("cannot marshal manifest %s", err.Error())
	}

	tpl, err := template.New("").Funcs(sprig.TxtFuncMap()).Parse(string(manifestString))
	if err != nil {
		return err
	}

	var templated bytes.Buffer
	err = tpl.Execute(&templated, vars)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(templated.Bytes(), m)
}

func HelmTemplate(m Manifest) (string, error) {
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

func CloneChartFromRepo(m Manifest, privateKeyPath string) (string, error) {
	gitAddress, err := giturl.ParseScp(m.Chart.Name)
	if err != nil {
		return "", fmt.Errorf("cannot parse chart's git address: %s", err)
	}
	gitUrl := strings.ReplaceAll(m.Chart.Name, gitAddress.RawQuery, "")
	gitUrl = strings.ReplaceAll(gitUrl, "?", "")

	tmpChartDir, err := ioutil.TempDir("", "gimlet-git-chart")
	if err != nil {
		return "", fmt.Errorf("cannot create tmp file: %s", err)
	}

	opts := &git.CloneOptions{
		URL: gitUrl,
	}
	if privateKeyPath != "" {
		publicKeys, err := ssh.NewPublicKeysFromFile("git", privateKeyPath, "")
		if err != nil {
			return "", fmt.Errorf("cannot generate public key from private: %s", err.Error())
		}
		opts.Auth = publicKeys
	}
	repo, err := git.PlainClone(tmpChartDir, false, opts)
	if err != nil {
		return "", fmt.Errorf("cannot clone chart git repo: %s", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("cannot get worktree: %s", err)
	}

	params, _ := url.ParseQuery(gitAddress.RawQuery)
	if v, found := params["path"]; found {
		tmpChartDir = tmpChartDir + v[0]
	}
	if v, found := params["sha"]; found {
		err = worktree.Checkout(&git.CheckoutOptions{
			Hash: plumbing.NewHash(v[0]),
		})
		if err != nil {
			return "", fmt.Errorf("cannot checkout sha: %s", err)
		}
	}
	if v, found := params["tag"]; found {
		err = worktree.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewTagReferenceName(v[0]),
		})
		if err != nil {
			return "", fmt.Errorf("cannot checkout tag: %s", err)
		}
	}
	if v, found := params["branch"]; found {
		err = worktree.Checkout(&git.CheckoutOptions{
			Branch: plumbing.NewBranchReferenceName(v[0]),
		})
		if err != nil {
			return "", fmt.Errorf("cannot checkout branch: %s", err)
		}
	}

	return tmpChartDir, nil
}
