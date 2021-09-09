package notifications

import (
	"fmt"
	"github.com/gimlet-io/gimletd/worker/events"
	githubLib "github.com/google/go-github/v37/github"
	"strings"
)

type gitopsRollbackMessage struct {
	event *events.RollbackEvent
}

func (gm *gitopsRollbackMessage) AsSlackMessage() (*slackMessage, error) {
	msg := &slackMessage{
		Text:   "",
		Blocks: []Block{},
	}

	if gm.event.Status == events.Failure {
		msg.Text = fmt.Sprintf("Failed to roll back %s of %s",
			gm.event.RollbackRequest.App,
			gm.event.RollbackRequest.Env)
		msg.Blocks = append(msg.Blocks,
			Block{
				Type: section,
				Text: &Text{
					Type: markdown,
					Text: msg.Text,
				},
			},
		)
		msg.Blocks = append(msg.Blocks,
			Block{
				Type: contextString,
				Elements: []Text{
					{
						Type: markdown,
						Text: fmt.Sprintf(":exclamation: *Error* :exclamation: \n%s", gm.event.StatusDesc),
					},
				},
			},
		)
		msg.Blocks = append(msg.Blocks,
			Block{
				Type: contextString,
				Elements: []Text{
					{Type: markdown, Text: fmt.Sprintf(":dart: %s", strings.Title(gm.event.RollbackRequest.Env))},
					{Type: markdown, Text: fmt.Sprintf(":clipboard: %s", gm.event.RollbackRequest.TargetSHA)},
				},
			},
		)
	} else {
		msg.Text = fmt.Sprintf("🔙 Rollback %s of %s", gm.event.RollbackRequest.App, gm.event.RollbackRequest.Env)
		msg.Blocks = append(msg.Blocks,
			Block{
				Type: section,
				Text: &Text{
					Type: markdown,
					Text: msg.Text,
				},
			},
		)
		msg.Blocks = append(msg.Blocks,
			Block{
				Type: contextString,
				Elements: []Text{
					{Type: markdown, Text: fmt.Sprintf(":dart: %s", strings.Title(gm.event.RollbackRequest.Env))},
					{Type: markdown, Text: fmt.Sprintf(":clipboard: %s", gm.event.RollbackRequest.TargetSHA)},
				},
			},
		)
		for _, gitopsRef := range gm.event.GitopsRefs {
			msg.Blocks[len(msg.Blocks)-1].Elements = append(
				msg.Blocks[len(msg.Blocks)-1].Elements,
				Text{Type: markdown, Text: fmt.Sprintf(":paperclip: %s", commitLink(gm.event.GitopsRepo, gitopsRef))},
			)
		}
		if len(msg.Blocks[len(msg.Blocks)-1].Elements) > 10 {
			msg.Blocks[len(msg.Blocks)-1].Elements = msg.Blocks[len(msg.Blocks)-1].Elements[:10]
		}
	}

	return msg, nil
}

func (gm *gitopsRollbackMessage) Env() string {
	return gm.event.RollbackRequest.Env
}

func (gm *gitopsRollbackMessage) AsGithubStatus() (*githubLib.RepoStatus, error) {
	return nil, nil
}

func MessageFromRollbackEvent(event *events.RollbackEvent) Message {
	return &gitopsRollbackMessage{
		event: event,
	}
}

func (gm *gitopsRollbackMessage) RepositoryName() string {
	return ""
}

func (gm *gitopsRollbackMessage) SHA() string {
	return ""
}
