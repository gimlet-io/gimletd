package githelper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gimlet-io/gimlet-cli/commands"
	"github.com/gimlet-io/gimletd/dx"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const gitSSHAddressFormat = "git@github.com:%s.git"

func CloneToTmpFs(repoName string, privateKeyPath string) (string, *git.Repository, error) {
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

func RemoteHasChanges(repo *git.Repository, privateKeyPath string) (bool, error) {
	publicKeys, err := ssh.NewPublicKeysFromFile("git", privateKeyPath, "")
	if err != nil {
		return false, fmt.Errorf("cannot generate public key from private: %s", err.Error())
	}

	err = repo.Fetch(&git.FetchOptions{
		Auth: publicKeys,
	})
	if err == git.NoErrAlreadyUpToDate {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func TmpFsCleanup(path string) error {
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

	return err
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

	_, err = worktree.Filesystem.Stat(path)
	if err != nil {
		return nil
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

func CommitFilesToGit(
	repo *git.Repository,
	files map[string]string,
	env string,
	app string,
	message string,
	releaseString string,
) (string, error) {
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

	// first delete, then recreate app dir
	// to remove stale template files
	err = DelDir(repo, filepath.Join(env, app))
	if err != nil {
		return "", fmt.Errorf("cannot del dir: %s", err)
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

	if releaseString != "" {
		if !strings.HasSuffix(releaseString, "\n") {
			releaseString = releaseString + "\n"
		}
		err = stageFile(w, releaseString, filepath.Join(env, "release.json"))
		if err != nil {
			return "", fmt.Errorf("cannot stage file %s", err)
		}
		err = stageFile(w, releaseString, filepath.Join(env, app, "release.json"))
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
	limit int,
	gitRepo string,
) ([]*dx.Release, error) {
	releases := []*dx.Release{}

	var path string
	if env == "" {
		return nil, fmt.Errorf("env is mandatory")
	} else {
		if app != "" {
			path = fmt.Sprintf("%s/%s", env, app)
		} else {
			path = env
		}
	}

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
		if limit != 0 && len(releases) >= limit {
			return fmt.Errorf("%s", "LIMIT")
		}

		if RollbackCommit(c) {
			return nil
		}

		releaseFile, err := c.File(env + "/release.json")
		if err != nil {
			releaseFile, err = c.File(path + "/release.json")
			if err != nil {
				logrus.Debugf("no release file for %s: %s", c.Hash.String(), err)
				releases = append(releases, relaseFromCommit(c, app, env))
				return nil
			}
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

		if gitRepo != "" { // gitRepo filter
			if release.Version.RepositoryName != gitRepo {
				return nil
			}
		}

		release.Created = c.Committer.When.Unix()
		release.GitopsRef = c.Hash.String()

		rolledBack, err := HasBeenReverted(repo, c.Hash.String(), env, app)
		if err != nil {
			logrus.Warnf("cannot determine if commit was rolled back %s: %s", c.Hash.String(), err)
			releases = append(releases, relaseFromCommit(c, app, env))
		}
		release.RolledBack = rolledBack

		releases = append(releases, release)

		return nil
	})
	if err != nil &&
		err.Error() != "EOF" &&
		err.Error() != "LIMIT" {
		return nil, err
	}

	return releases, nil
}

func Status(
	repo *git.Repository,
	app, env string,
	perf *prometheus.HistogramVec,
) (map[string]*dx.Release, error) {
	t0 := time.Now()
	appReleases := map[string]*dx.Release{}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	fs := worktree.Filesystem

	if env == "" {
		return nil, fmt.Errorf("env is mandatory")
	} else {
		if app != "" {
			path := filepath.Join(env, app)
			release, err := readAppStatus(fs, path)
			if err != nil {
				return nil, fmt.Errorf("cannot read app status %s: %s", path, err)
			}

			appReleases[app] = release
		} else {
			paths, err := fs.ReadDir(env)
			if err != nil {
				return nil, fmt.Errorf("cannot list files: %s", err)
			}

			for _, fileInfo := range paths {
				if !fileInfo.IsDir() {
					continue
				}
				path := filepath.Join(env, fileInfo.Name())

				release, err := readAppStatus(fs, path)
				if err != nil {
					logrus.Debugf("cannot read app status %s: %s", path, err)
				}

				appReleases[fileInfo.Name()] = release
			}
		}
	}

	logrus.Infof("githelper_status: %f", time.Since(t0).Seconds())
	perf.WithLabelValues("githelper_status").Observe(time.Since(t0).Seconds())
	return appReleases, nil
}

func Envs(
	repo *git.Repository,
) ([]string, error) {
	var envs []string

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	fs := worktree.Filesystem

	paths, err := fs.ReadDir("/")
	if err != nil {
		return nil, fmt.Errorf("cannot list files: %s", err)
	}

	for _, fileInfo := range paths {
		if !fileInfo.IsDir() {
			continue
		}

		dir := fileInfo.Name()
		_, err := readAppStatus(fs, dir)
		if err == nil {
			envs = append(envs, dir)
		}
	}

	return envs, nil
}

func readAppStatus(fs billy.Filesystem, path string) (*dx.Release, error) {
	var release *dx.Release
	f, err := fs.Open(path + "/release.json")
	if err != nil {
		return nil, err
	}

	releaseBytes, err := ioutil.ReadAll(f)
	err = json.Unmarshal(releaseBytes, &release)
	defer f.Close()
	return release, err
}

func RollbackCommit(c *object.Commit) bool {
	return strings.Contains(c.Message, "This reverts commit")
}

func HasBeenReverted(repo *git.Repository, sha string, env string, app string) (bool, error) {
	path := fmt.Sprintf("%s/%s", env, app)
	commits, err := repo.Log(
		&git.LogOptions{
			Path: &path,
		},
	)
	if err != nil {
		return false, errors.WithMessage(err, "could not walk commits")
	}

	hasBeenReverted := false
	err = commits.ForEach(func(c *object.Commit) error {
		if strings.Contains(c.Message, sha) {
			hasBeenReverted = true
			return fmt.Errorf("EOF")
		}
		return nil
	})
	if err != nil && err.Error() != "EOF" {
		return false, err
	}

	return hasBeenReverted, nil
}

func relaseFromCommit(c *object.Commit, app string, env string) *dx.Release {
	return &dx.Release{
		App:       app,
		Env:       env,
		Created:   c.Committer.When.Unix(),
		GitopsRef: c.Hash.String(),
	}
}
