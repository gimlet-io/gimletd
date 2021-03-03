package githelper

import (
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Releases(t *testing.T) {
	repo := initHistory()

	releases, err := Releases(repo, "my-app", "staging", nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(releases),"should get all releases")
}

func initHistory() *git.Repository {
	repo, _ := git.Init(memory.NewStorage(), memfs.New())

	CommitFilesToGit(
		repo,
		map[string]string{
			"release.json": `{"app":"fosdem-2021","env":"staging","artifactId":"my-app-94578d91-ef9a-413d-9afb-602256d2b124","repositoryName":"laszlocph/gimletd-test","sha":"d7aa20d7055999200b52c4ffd146d5c7c415e3e7","branch":"master","triggeredBy":"policy","gitopsRef":"","gitopsRepo":""}`,
		},
		"staging",
		"my-app2",
		"1st commit",
	)
	CommitFilesToGit(
		repo,
		map[string]string{
			"release.json": `{"app":"fosdem-2021","env":"staging","artifactId":"my-app-94578d91-ef9a-413d-9afb-602256d2b124","repositoryName":"laszlocph/gimletd-test","sha":"d7aa20d7055999200b52c4ffd146d5c7c415e3e7","branch":"master","triggeredBy":"policy","gitopsRef":"","gitopsRepo":""}`,
		},
		"staging",
		"my-app",
		"1st commit",
	)
	CommitFilesToGit(
		repo,
		map[string]string{
			"release.json": `{"app":"fosdem-2022","env":"staging","artifactId":"my-app-94578d91-ef9a-413d-9afb-602256d2b124","repositoryName":"laszlocph/gimletd-test","sha":"d7aa20d7055999200b52c4ffd146d5c7c415e3e7","branch":"master","triggeredBy":"policy","gitopsRef":"","gitopsRepo":""}`,
		},
		"staging",
		"my-app",
		"2nd commit",
	)
	CommitFilesToGit(
		repo,
		map[string]string{
			"release.json": `{"app":"fosdem-2023","env":"staging","artifactId":"my-app-94578d91-ef9a-413d-9afb-602256d2b124","repositoryName":"laszlocph/gimletd-test","sha":"d7aa20d7055999200b52c4ffd146d5c7c415e3e7","branch":"master","triggeredBy":"policy","gitopsRef":"","gitopsRepo":""}`,
		},
		"staging",
		"my-app",
		"3rd commit",
	)

	return repo
}
