// Copyright 2019 Laszlo Fogas
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package worker

import (
	"encoding/json"
	"github.com/gimlet-io/gimletd/artifact"
	"github.com/gimlet-io/gimletd/manifest"
	"github.com/gimlet-io/gimletd/model"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_process(t *testing.T) {
	var a artifact.Artifact
	json.Unmarshal([]byte(`
{
  "version": {
    "repositoryName": "my-app",
    "sha": "ea9ab7cc31b2599bf4afcfd639da516ca27a4780",
    "branch": "master",
    "authorName": "Jane Doe",
    "authorEmail": "jane@doe.org",
    "committerName": "Jane Doe",
    "committerEmail": "jane@doe.org",
    "message": "Bugfix 123",
    "url": "https://github.com/gimlet-io/gimlet-cli/commit/ea9ab7cc31b2599bf4afcfd639da516ca27a4780"
  },
  "environments": [
    {
      "App": "my-app",
      "Env": "staging",
      "Namespace": "staging",
      "Deploy": {
        "Branch": "master",
        "Event": "push"
      },
      "Chart": {
        "Repository": "https://chart.onechart.dev",
        "Name": "onechart",
        "Version": "0.10.0"
      },
      "Values": {
        "image": {
          "repository": "ghcr.io/gimlet-io/my-app",
          "tag": "{{ .GITHUB_SHA }}"
        },
        "replicas": 1
      }
    }
  ],
  "items": [
    {
      "name": "CI",
      "url": "https://jenkins.example.com/job/dev/84/display/redirect"
    }
  ]
}
`), &a)
	artifactModel, _ := model.ToArtifactModel(a)

	err := process(artifactModel)
	assert.Nil(t, err)
}

func Test_deployTrigger(t *testing.T) {
	triggered := deployTrigger(
		&artifact.Artifact{}, nil)
	assert.False(t, triggered, "Empty deploy policy should not trigger a deploy")

	triggered = deployTrigger(
		&artifact.Artifact{}, &manifest.Deploy{})
	assert.False(t, triggered, "Empty deploy policy should not trigger a deploy")

	triggered = deployTrigger(
		&artifact.Artifact{
			Version: artifact.Version{
				Branch: "master",
			},
		},
		&manifest.Deploy{
			Branch: "notMaster",
		})
	assert.False(t, triggered, "Branch mismatch should not trigger a deploy")

	triggered = deployTrigger(
		&artifact.Artifact{
			Version: artifact.Version{
				Branch: "master",
			},
		},
		&manifest.Deploy{
			Branch: "master",
		})
	assert.True(t, triggered, "Matching branch should trigger a deploy")

	triggered = deployTrigger(
		&artifact.Artifact{},
		&manifest.Deploy{
			Event: manifest.PushEvent,
		})
	assert.False(t, triggered, "Not yet supported")

	triggered = deployTrigger(
		&artifact.Artifact{Version: artifact.Version{

		}},
		&manifest.Deploy{
			Event: manifest.PREvent,
		})
	assert.False(t, triggered, "Not matching PR event should not trigger a deploy")

	triggered = deployTrigger(
		&artifact.Artifact{Version: artifact.Version{
			PR: true,
		}},
		&manifest.Deploy{
			Event: manifest.PREvent,
		})
	assert.True(t, triggered, "Should trigger a PR deploy")
}
