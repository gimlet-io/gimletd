package worker

import (
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"strings"
	"testing"
)

type dummyTokenManager struct {
}

func (t *dummyTokenManager) Token() (string, string, error) {
	return "xxx", "", nil
}

func Test_deletedBranches(t *testing.T) {
	//os.RemoveAll("/tmp/gimletd")
	//os.MkdirAll("/tmp/gimletd", 0777)

	branchDeleteEventWorker := NewBranchDeleteEventWorker(
		&dummyTokenManager{},
		"/tmp/gimletd",
		nil)

	//branchDeleteEventWorker.clone("laszlocph/gimletd-test-repo")

	repoPath := filepath.Join("/tmp/gimletd", strings.ReplaceAll("laszlocph/gimletd-test-repo", "/", "%"))
	repo, err := git.PlainOpen(repoPath)
	assert.Nil(t, err)
	deletedBranches := branchDeleteEventWorker.detectDeletedBranches(repo)
	fmt.Println(deletedBranches)
}
