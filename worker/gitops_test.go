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
	"fmt"
	"github.com/gimlet-io/gimletd/dx"
	"github.com/gimlet-io/gimletd/githelper"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func Test_gitopsTemplateAndWrite(t *testing.T) {
	var a dx.Artifact
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

	repo, _ := git.Init(memory.NewStorage(), memfs.New())
	_, err := repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{""}})

	_, err = gitopsTemplateAndWrite(repo, a.Context, a.Environments[0], &dx.Release{}, "")
	assert.Nil(t, err)
}

func Test_deployTrigger(t *testing.T) {
	triggered := deployTrigger(
		&dx.Artifact{}, nil)
	assert.False(t, triggered, "Empty deploy policy should not trigger a deploy")

	triggered = deployTrigger(
		&dx.Artifact{}, &dx.Deploy{})
	assert.False(t, triggered, "Empty deploy policy should not trigger a deploy")

	triggered = deployTrigger(
		&dx.Artifact{
			Version: dx.Version{
				Branch: "master",
			},
		},
		&dx.Deploy{
			Branch: "notMaster",
		})
	assert.False(t, triggered, "Branch mismatch should not trigger a deploy")

	triggered = deployTrigger(
		&dx.Artifact{
			Version: dx.Version{
				Branch: "master",
			},
		},
		&dx.Deploy{
			Branch: "master",
		})
	assert.True(t, triggered, "Matching branch should trigger a deploy")

	triggered = deployTrigger(
		&dx.Artifact{},
		&dx.Deploy{
			Event: dx.PushPtr(),
		})
	assert.True(t, triggered, "Default Push event should trigger a deploy")

	triggered = deployTrigger(
		&dx.Artifact{},
		&dx.Deploy{},
	)
	assert.False(t, triggered, "Non matching event should not trigger a deploy, default is Push in the Artifact")

	triggered = deployTrigger(
		&dx.Artifact{},
		&dx.Deploy{
			Event: dx.PRPtr(),
		})
	assert.False(t, triggered, "Non matching event should not trigger a deploy")

	triggered = deployTrigger(
		&dx.Artifact{Version: dx.Version{
			Event: dx.PR,
		}},
		&dx.Deploy{
			Event: dx.PRPtr(),
		})
	assert.True(t, triggered, "Should trigger a PR deploy")

	triggered = deployTrigger(
		&dx.Artifact{Version: dx.Version{
			Event: dx.Tag,
		}},
		&dx.Deploy{
			Event: dx.TagPtr(),
		})
	assert.True(t, triggered, "Should trigger a PR deploy")
}

func Test_revertTo(t *testing.T) {
	path, _ := ioutil.TempDir("", "gitops-")
	defer os.RemoveAll(path)

	repo, _ := git.PlainInit(path, false)
	initHistory(repo)

	var SHAs []string
	commits, _ := repo.Log(&git.LogOptions{})
	 commits.ForEach(func(c *object.Commit) error {
		SHAs = append(SHAs, c.Hash.String())
		return nil
	})

	err := revertTo(
		"staging",
		"my-app",
		repo,
		path,
		SHAs[2],
	)
	assert.Nil(t, err)
	content, _ := githelper.Content(repo, "staging/my-app/file")
	assert.Equal(t, "1\n", content)

	SHAs = []string{}
	commits, _ = repo.Log(&git.LogOptions{})
	commits.ForEach(func(c *object.Commit) error {
		SHAs = append(SHAs, c.Hash.String())
		return nil
	})
	assert.Equal(t, 6, len(SHAs))

	err = revertTo(
		"staging",
		"my-app",
		repo,
		path,
		SHAs[4],
	)
	assert.Nil(t, err)
	content, _ = githelper.Content(repo, "staging/my-app/file")
	assert.Equal(t, "1\n", content)

	err = revertTo(
		"staging",
		"my-app",
		repo,
		path,
		SHAs[5],
	)
	assert.Nil(t, err)
	content, _ = githelper.Content(repo, "staging/my-app/file")
	assert.Equal(t, "0\n", content)
}

func initHistory(repo *git.Repository) {
	sha, _ := githelper.CommitFilesToGit(
		repo,
		map[string]string{
			"file": `0`,
		},
		"staging",
		"my-app",
		"0st commit",
	)
	fmt.Printf("%s - %s\n", sha, "0")
	sha, _ = githelper.CommitFilesToGit(
		repo,
		map[string]string{
			"file": `1`,
		},
		"staging",
		"my-app",
		"1st commit",
	)
	fmt.Printf("%s - %s\n", sha, "1")
	sha, _ = githelper.CommitFilesToGit(
		repo,
		map[string]string{
			"file": `2`,
		},
		"staging",
		"my-app",
		"2nd commit",
	)
	fmt.Printf("%s - %s\n", sha, "2")
	sha, _ = githelper.CommitFilesToGit(
		repo,
		map[string]string{
			"file": `3`,
		},
		"staging",
		"my-app",
		"3rd commit",
	)
	fmt.Printf("%s - %s\n", sha, "3")
}
