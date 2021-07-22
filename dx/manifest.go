package dx

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"regexp"
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
	Tag    string    `yaml:"tag,omitempty" json:"tag,omitempty"`
	Branch string    `yaml:"branch,omitempty" json:"branch,omitempty"`
	Event  *GitEvent `yaml:"event,omitempty" json:"event,omitempty"`
}

func (m *Manifest) ResolveVars(vars map[string]string) error {
	manifestString, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("cannot marshal manifest %s", err.Error())
	}

	functions := make(map[string]interface{})
	for k, v := range sprig.GenericFuncMap() {
		functions[k] = v
	}
	functions["sanitizeDNSName"] = sanitizeDNSName
	tpl, err := template.New("").
		Funcs(functions).
		Parse(string(manifestString))
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

func sanitizeDNSName(str string) string {
	r := regexp.MustCompile("[^0-9A-Za-z_\\\\.]+")
	str = r.ReplaceAllString(str, "-")
	str = strings.ToLower(str)
	if len(str) > 63 {
		str = str[0:63]
	}
	str = strings.TrimSuffix(str, "-")
	return str
}
