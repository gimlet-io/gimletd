package notifications

import (
	"fmt"
	"strings"

	"github.com/gimlet-io/gimletd/worker/events"
	githubLib "github.com/google/go-github/v37/github"
)

type gitopsDeleteMessage struct {
	event *events.DeleteEvent
}

func (gm *gitopsDeleteMessage) AsSlackMessage() (*slackMessage, error) {
	msg := &slackMessage{
		Text:   "",
		Blocks: []Block{},
	}

	if gm.event.Status == events.Failure {
		msg.Text = fmt.Sprintf("Failed to delete %s of %s", gm.event.App, gm.event.Env)
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
	} else {
		msg.Text = fmt.Sprintf("Deleting %s of %s", gm.event.App, gm.event.Env)
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
					{Type: markdown, Text: fmt.Sprintf(":dart: %s", strings.Title(gm.event.Env))},
					{Type: markdown, Text: fmt.Sprintf(":paperclip: %s", commitLink(gm.event.GitopsRepo, gm.event.GitopsRef))},
				},
			},
		)
	}

	return msg, nil
}

func (gm *gitopsDeleteMessage) Env() string {
	return gm.event.Env
}

func (gm *gitopsDeleteMessage) AsGithubStatus() (*githubLib.RepoStatus, error) {
	return nil, nil
}

func MessageFromDeleteEvent(event *events.DeleteEvent) Message {
	return &gitopsDeleteMessage{
		event: event,
	}
}

func (gm *gitopsDeleteMessage) RepositoryName() string {
	return ""
}

func (gm *gitopsDeleteMessage) SHA() string {
	return ""
}
