package dx

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_vars(t *testing.T) {
	var a Artifact
	json.Unmarshal([]byte(`
{
  "version": {},
  "environments": [],
  "context": {
	"CI_VAR": "civalue"
  },
  "items": [
    {
      "name": "image",
      "url": "nginx"
    }
  ]
}
`), &a)

	vars := a.Vars()
	assert.Equal(t, 3, len(vars))
	assert.Equal(t, 1, len(a.Context))
}
