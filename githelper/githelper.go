package githelper

import (
	"fmt"
	"github.com/gimlet-io/gimlet-cli/commands"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"
)

const gitSSHAddressFormat = "git@github.com:%s.git"

// CloneToMemory checks out a repo to an in-memory filesystem
func CloneToMemory(repoName string, privateKeyPath string) (*git.Repository, error) {
	url := fmt.Sprintf(gitSSHAddressFormat, repoName)
	publicKeys, err := ssh.NewPublicKeysFromFile("git", privateKeyPath, "")
	if err != nil {
		return nil, fmt.Errorf("cannot generate public key from private: %s", err.Error())
	}

	fs := memfs.New()
	repo, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL:   url,
		Depth: 1,
		Auth:  publicKeys,
	})

	if err != nil && strings.Contains(err.Error(), "remote repository is empty") {
		repo, _ := git.Init(memory.NewStorage(), memfs.New())
		_, err = repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{url}})
		return repo, err
	}

	return repo, err
}

func Push(repo *git.Repository, privateKeyPath string) error {
	publicKeys, err := ssh.NewPublicKeysFromFile("git", privateKeyPath, "")
	if err != nil {
		return fmt.Errorf("cannot generate public key from private: %s", err.Error())
	}

	err = repo.Push(&git.PushOptions{
		Auth:  publicKeys,
	})

	if err == git.NoErrAlreadyUpToDate {
		return nil
	}

	return nil
}

func NothingToCommit(repo *git.Repository) (bool, error) {
	worktree, err := repo.Worktree()
	if err != nil {
		return false, err
	}

	status, err := worktree.Status()
	if err != nil {
		return false, err
	}

	return status.IsClean(), nil
}

func Commit(repo *git.Repository, message string) (string, error) {
	worktree, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	sha, err := worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Gimlet CLI",
			Email: "cli@gimlet.io",
			When:  time.Now(),
		},
	})

	if err != nil {
		return "", err
	}

	return sha.String(), nil
}

func DelDir(repo *git.Repository, path string) error {
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	files, err := worktree.Filesystem.ReadDir(path)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			DelDir(repo, file.Name())
		}

		_, err = worktree.Remove(filepath.Join(path, file.Name()))
		if err != nil {
			return err
		}
	}

	_, err = worktree.Remove(path)

	return err
}

func StageFolder(repo *git.Repository, folder string) error {
	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	return worktree.AddWithOptions(&git.AddOptions{
		Glob: folder + "/*",
	})
}

func CommitFilesToGit(repo *git.Repository, files map[string]string, env string, app string, message string) (string, error) {
	empty, err := NothingToCommit(repo)
	if err != nil {
		return "", fmt.Errorf("cannot get git state %s", err)
	}
	if !empty {
		return "", fmt.Errorf("there are staged changes in the gitops repo. Commit them first then try again")
	}

	w, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("cannot get worktree %s", err)
	}
	err = w.Filesystem.MkdirAll(filepath.Join(env, app), commands.Dir_RWX_RX_R)
	if err != nil {
		return "", fmt.Errorf("cannot create dir %s", err)
	}

	for path, content := range files {
		if !strings.HasSuffix(content, "\n") {
			content = content + "\n"
		}

		err = stageFile(w, content, filepath.Join(env, app, filepath.Base(path)))
		if err != nil {
			return "", fmt.Errorf("cannot stage file %s", err)
		}
	}

	empty, err = NothingToCommit(repo)
	if err != nil {
		return "", err
	}
	if empty {
		return "", nil
	}

	gitMessage := fmt.Sprintf("[Gimlet] %s/%s %s", env, app, message)
	return Commit(repo, gitMessage)
}

func stageFile(worktree *git.Worktree, content string, path string) error {
	createdFile, err := worktree.Filesystem.Create(path)
	if err != nil {
		return err
	}
	_, err = createdFile.Write([]byte(content))
	if err != nil {
		return err
	}
	err = createdFile.Close()

	_, err = worktree.Add(path)
	return err
}

// Content returns the content of a file
func Content(repo *git.Repository, path string) (string, error) {
	worktree, err := repo.Worktree()
	if err != nil {
		return "", err
	}

	f, err := worktree.Filesystem.Open(path)
	if err != nil {
		return "", nil
	}
	defer f.Close()

	content, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
