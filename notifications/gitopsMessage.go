package notifications

import (
	"fmt"
	"strings"
)

type gitopsMessage struct {
	event *GitopsEvent
}

func (gm *gitopsMessage) AsSlackMessage() (*slackMessage, error) {
	msg := &slackMessage{
		Text:   "",
		Blocks: []Block{},
	}

	if gm.event.Status == Failure {
		msg.Text = fmt.Sprintf("Failed to roll out %s of %s", gm.event.Manifest.App, gm.event.Artifact.Version.RepositoryName)
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
					{Type: markdown, Text: fmt.Sprintf(":dart: %s", strings.Title(gm.event.Manifest.Env))},
					{Type: markdown, Text: fmt.Sprintf(":clipboard: %s", gm.event.Artifact.Version.URL)},
				},
			},
		)
	} else {
		msg.Text = fmt.Sprintf("Rolling out %s of %s", gm.event.Manifest.App, gm.event.Artifact.Version.RepositoryName)
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
					{Type: markdown, Text: fmt.Sprintf(":dart: %s", strings.Title(gm.event.Manifest.Env))},
					{Type: markdown, Text: fmt.Sprintf(":clipboard: %s", gm.event.Artifact.Version.URL)},
					{Type: markdown, Text: fmt.Sprintf(":paperclip: %s", commitLink(gm.event.GitopsRepo, gm.event.GitopsRef))},
				},
			},
		)
	}

	return msg, nil
}

func (gm *gitopsMessage) Env() string {
	return gm.event.Manifest.Env
}

func MessageFromGitOpsEvent(event *GitopsEvent) Message {
	return &gitopsMessage{
		event: event,
	}
}
