package notifications

import (
	"fmt"
	"strings"
	"time"

	"github.com/gimlet-io/gimletd/worker/events"
	githubLib "github.com/google/go-github/v37/github"
)

const githubCommitLink = "https://github.com/%s/commit/%s"
const contextFormat = "gitops/%s@%s"

type gitopsDeployMessage struct {
	event *events.DeployEvent
}

func (gm *gitopsDeployMessage) AsSlackMessage() (*slackMessage, error) {
	msg := &slackMessage{
		Text:   "",
		Blocks: []Block{},
	}

	if gm.event.Status == events.Failure {
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
		msg.Text = fmt.Sprintf("%s is rolling out %s on %s", gm.event.TriggeredBy, gm.event.Manifest.App, gm.event.Artifact.Version.RepositoryName)
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

func (gm *gitopsDeployMessage) Env() string {
	return gm.event.Manifest.Env
}

func (gm *gitopsDeployMessage) AsGithubStatus() (*githubLib.RepoStatus, error) {
	context := fmt.Sprintf(contextFormat, gm.event.Manifest.Env, time.Now().Format(time.RFC3339))
	desc := gm.event.StatusDesc
	if len(desc) > 140 {
		desc = desc[:140]
	}

	state := "success"
	targetURL := fmt.Sprintf(githubCommitLink, gm.event.GitopsRepo, gm.event.GitopsRef)
	targetURLPtr := &targetURL

	if gm.event.Status == events.Failure {
		state = "failure"
		targetURLPtr = nil
	}

	return &githubLib.RepoStatus{
		State:       &state,
		Context:     &context,
		Description: &desc,
		TargetURL:   targetURLPtr,
	}, nil
}

func MessageFromGitOpsEvent(event *events.DeployEvent) Message {
	return &gitopsDeployMessage{
		event: event,
	}
}

func (gm *gitopsDeployMessage) RepositoryName() string {
	return gm.event.Artifact.Version.RepositoryName
}

func (gm *gitopsDeployMessage) SHA() string {
	return gm.event.Artifact.Version.SHA
}
