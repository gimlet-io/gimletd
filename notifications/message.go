package notifications

import githubLib "github.com/google/go-github/v37/github"

type Message interface {
	AsSlackMessage(SendProgressingMessages bool) (*slackMessage, error)
	AsGithubStatus() (*githubLib.RepoStatus, error)
	Env() string
	RepositoryName() string
	SHA() string
}
