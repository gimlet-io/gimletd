package notifications

import githubLib "github.com/google/go-github/v33/github"

type Message interface {
	AsSlackMessage() (*slackMessage, error)
	AsGithubStatus() (*githubLib.RepoStatus, error)
	Env() string
	RepositoryName() string
	SHA() string
}
