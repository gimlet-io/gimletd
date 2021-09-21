package notifications

import (
	"fmt"
	"github.com/gimlet-io/gimletd/model"
	githubLib "github.com/google/go-github/v37/github"
)

type fluxMessage struct {
	gitopsCommit *model.GitopsCommit
	gitopsRepo   string
	env          string
}

func (fm *fluxMessage) AsSlackMessage() (*slackMessage, error) {
	if fm.gitopsCommit.Status == model.Progressing {
		return nil, nil
	}

	msg := &slackMessage{
		Text:   "",
		Blocks: []Block{},
	}

	if fm.gitopsCommit.Status == model.ReconciliationSucceeded {
		msg.Text = fmt.Sprintf("Gitops changes applied :heavy_check_mark: %s", commitLink(fm.gitopsRepo, fm.gitopsCommit.Sha))
	}

	if fm.gitopsCommit.Status == model.ValidationFailed ||
		fm.gitopsCommit.Status == model.ReconciliationFailed {
		msg.Text = ":exclamation: Gitops apply failed"
	}

	msg.Blocks = append(msg.Blocks,
		Block{
			Type: section,
			Text: &Text{
				Type: markdown,
				Text: msg.Text,
			},
		},
	)

	if fm.gitopsCommit.Status == model.ValidationFailed ||
		fm.gitopsCommit.Status == model.ReconciliationFailed {
		msg.Blocks = append(msg.Blocks,
			Block{
				Type: contextString,
				Elements: []Text{
					{
						Type: markdown,
						Text: fm.gitopsCommit.StatusDesc,
					},
				},
			},
		)
	}

	return msg, nil
}

func (fm *fluxMessage) Env() string {
	return fm.env
}

func (fm *fluxMessage) AsGithubStatus() (*githubLib.RepoStatus, error) {
	return nil, nil
}

func NewMessage(gitopsRepo string, gitopsCommit *model.GitopsCommit, env string) Message {
	return &fluxMessage{
		gitopsCommit: gitopsCommit,
		gitopsRepo:   gitopsRepo,
		env:          env,
	}
}

func (fm *fluxMessage) RepositoryName() string {
	return ""
}

func (fm *fluxMessage) SHA() string {
	return ""
}
