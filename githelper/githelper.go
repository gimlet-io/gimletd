package githelper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gimlet-io/gimlet-cli/commands"
	"github.com/gimlet-io/gimletd/dx"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const gitSSHAddressFormat = "git@github.com:%s.git"

// CloneToMemory checks out a repo to an in-memory filesystem
func CloneToMemory(repoName string, privateKeyPath string, shallow bool) (*git.Repository, error) {
	url := fmt.Sprintf(gitSSHAddressFormat, repoName)
	publicKeys, err := ssh.NewPublicKeysFromFile("git", privateKeyPath, "")
	if err != nil {
		return nil, fmt.Errorf("cannot generate public key from private: %s", err.Error())
	}

	fs := memfs.New()
	opts := &git.CloneOptions{
		URL:  url,
		Auth: publicKeys,
	}
	if shallow {
		opts.Depth = 1
	}
	repo, err := git.Clone(memory.NewStorage(), fs, opts)

	if err != nil && strings.Contains(err.Error(), "remote repository is empty") {
		repo, _ := git.Init(memory.NewStorage(), memfs.New())
		_, err = repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{url}})
		return repo, err
	}

	return repo, err
}

func NativeCheckout(repoName string, privateKeyPath string) (string, *git.Repository, error) {
	path, err := ioutil.TempDir("", "gitops-")
	if err != nil {
		errors.WithMessage(err, "get temporary directory")
	}
	url := fmt.Sprintf(gitSSHAddressFormat, repoName)
	publicKeys, err := ssh.NewPublicKeysFromFile("git", privateKeyPath, "")
	if err != nil {
		return "", nil, fmt.Errorf("cannot generate public key from private: %s", err.Error())
	}

	opts := &git.CloneOptions{
		URL:  url,
		Auth: publicKeys,
	}

	repo, err := git.PlainClone(path, false, opts)
	return path, repo, err
}

func NativeCleanup(path string) error {
	return os.RemoveAll(path)
}

func Push(repo *git.Repository, privateKeyPath string) error {
	publicKeys, err := ssh.NewPublicKeysFromFile("git", privateKeyPath, "")
	if err != nil {
		return fmt.Errorf("cannot generate public key from private: %s", err.Error())
	}

	err = repo.Push(&git.PushOptions{
		Auth: publicKeys,
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

func NativeRevert(repoPath string, sha string) error {
	return execCommand(repoPath, "git", "revert", sha)
}

func execCommand(rootPath string, cmdName string, args ...string) error {
	cmd := exec.CommandContext(context.TODO(), cmdName, args...)
	cmd.Dir = rootPath
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return errors.WithMessage(err, "get stdout pipe for command")
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return errors.WithMessage(err, "get stderr pipe for command")
	}
	err = cmd.Start()
	if err != nil {
		return errors.WithMessage(err, "start command")
	}

	stdoutData, err := ioutil.ReadAll(stdout)
	if err != nil {
		return errors.WithMessage(err, "read stdout data of command")
	}
	stderrData, err := ioutil.ReadAll(stderr)
	if err != nil {
		return errors.WithMessage(err, "read stderr data of command")
	}

	err = cmd.Wait()
	logrus.Infof("git/commit: exec command '%s %s': stdout: %s", cmdName, strings.Join(args, " "), stdoutData)
	logrus.Infof("git/commit: exec command '%s %s': stderr: %s", cmdName, strings.Join(args, " "), stderrData)
	if err != nil {
		return errors.WithMessage(err, "execute command failed")
	}

	if len(stderrData) != 0 {
		return errors.New(string(stderrData))
	}

	return nil
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

func Releases(
	repo *git.Repository,
	app, env string,
	since, until *time.Time,
) ([]*dx.Release, error) {
	releases := []*dx.Release{}

	path := fmt.Sprintf("%s/%s", env, app)
	commits, err := repo.Log(
		&git.LogOptions{
			Path:  &path,
			Since: since,
			Until: until,
		},
	)
	if err != nil {
		return nil, err
	}

	err = commits.ForEach(func(c *object.Commit) error {
		releaseFile, err := c.File(path + "/release.json")
		if err != nil {
			logrus.Debugf("no release file for %s: %s", c.Hash.String(), err)
			releases = append(releases, relaseFromCommit(c, app, env))
			return nil
		}

		buf := new(bytes.Buffer)
		reader, err := releaseFile.Blob.Reader()
		if err != nil {
			logrus.Warnf("cannot parse release file for %s: %s", c.Hash.String(), err)
			releases = append(releases, relaseFromCommit(c, app, env))
			return nil
		}

		buf.ReadFrom(reader)
		releaseBytes := buf.Bytes()

		var release *dx.Release
		err = json.Unmarshal(releaseBytes, &release)
		if err != nil {
			logrus.Warnf("cannot parse release file for %s: %s", c.Hash.String(), err)
			releases = append(releases, relaseFromCommit(c, app, env))
		}
		release.Created = c.Committer.When.Unix()
		release.GitopsRef = c.Hash.String()
		releases = append(releases, release)

		return nil
	})
	if err != nil && err.Error() != "EOF" {
		return nil, err
	}

	return releases, nil
}

func relaseFromCommit(c *object.Commit, app string, env string) *dx.Release {
	return &dx.Release{
		App:       app,
		Env:       env,
		Created:   c.Committer.When.Unix(),
		GitopsRef: c.Hash.String(),
	}
}
