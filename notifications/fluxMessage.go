package notifications

import (
	"fmt"
	"strings"

	"github.com/gimlet-io/gimletd/model"
	githubLib "github.com/google/go-github/v37/github"
)

type fluxMessage struct {
	gitopsCommit *model.GitopsCommit
	gitopsRepo   string
	env          string
}

func (fm *fluxMessage) AsSlackMessage(sendProgressingMessages bool) (*slackMessage, error) {
	if fm.gitopsCommit.Status == model.Progressing &&
		!sendProgressingMessages {
		return nil, nil
	}

	msg := &slackMessage{
		Text:   "",
		Blocks: []Block{},
	}

	switch fm.gitopsCommit.Status {
	case model.Progressing:
		msg.Text = fmt.Sprintf(":hourglass_flowing_sand: Applying gitops changes from %s", commitLink(fm.gitopsRepo, fm.gitopsCommit.Sha))
	case model.ReconciliationSucceeded:
		msg.Text = fmt.Sprintf(":heavy_check_mark: Gitops changes applied from %s", commitLink(fm.gitopsRepo, fm.gitopsCommit.Sha))
	case model.ValidationFailed:
	case model.ReconciliationFailed:
		msg.Text = fmt.Sprintf(":exclamation: Gitops changes from %s failed to apply", commitLink(fm.gitopsRepo, fm.gitopsCommit.Sha))
	case model.HealthCheckFailed:
		msg.Text = fmt.Sprintf(":ambulance: Gitops changes from %s have health issues", commitLink(fm.gitopsRepo, fm.gitopsCommit.Sha))
	default:
		msg.Text = fmt.Sprintf("%s: %s", fm.gitopsCommit.Status, commitLink(fm.gitopsRepo, fm.gitopsCommit.Sha))
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

	var contextText string
	switch fm.gitopsCommit.Status {
	case model.ValidationFailed:
	case model.ReconciliationFailed:
	case model.HealthCheckFailed:
		contextText = fm.gitopsCommit.StatusDesc
	case model.Progressing:
		if strings.Contains(fm.gitopsCommit.StatusDesc, "Health check passed") {
			contextText = fm.gitopsCommit.StatusDesc
		}
	}

	if contextText != "" {
		msg.Blocks = append(msg.Blocks,
			Block{
				Type: contextString,
				Elements: []Text{
					{
						Type: markdown,
						Text: contextText,
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
