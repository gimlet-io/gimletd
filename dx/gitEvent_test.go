package dx

import (
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
	"testing"
)

func Test_GitEventYaml(t *testing.T) {
	yamlStr := `
branch: main
event: pr
`
	var deployTrigger Deploy
	err := yaml.Unmarshal([]byte(yamlStr), &deployTrigger)
	assert.Nil(t, err)
	assert.True(t, deployTrigger.Branch == "main", "should parse branch")
	assert.True(t, *deployTrigger.Event == PR, "should parse event")
}
