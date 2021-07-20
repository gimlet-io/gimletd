package notifications

import (
	"fmt"
	"github.com/fluxcd/pkg/runtime/events"
	githubLib "github.com/google/go-github/v33/github"
	"strings"
)

type fluxMessage struct {
	event      *events.Event
	gitopsRepo string
}

func (fm *fluxMessage) AsSlackMessage() (*slackMessage, error) {
	if fm.event.Reason == "Progressing" {
		return nil, nil
	}

	msg := &slackMessage{
		Text:   "",
		Blocks: []Block{},
	}

	if fm.event.Reason == "ReconciliationSucceeded" {
		rev := ""
		if val, ok := fm.event.Metadata["revision"]; ok {
			rev = val
		}
		msg.Text = fmt.Sprintf("Gitops changes applied :heavy_check_mark: %s", commitLink(fm.gitopsRepo, parseRev(rev)))
	}

	if fm.event.Reason == "ValidationFailed" {
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

	//if fm.event.Reason == "ReconciliationSucceeded" {
	//	msg.Blocks = append(msg.Blocks,
	//		Block{
	//			Type: contextString,
	//			Elements: []Text{
	//				{
	//					Type: markdown,
	//					Text: fm.event.Message,
	//				},
	//			},
	//		},
	//	)
	//}

	if fm.event.Reason == "ValidationFailed" {
		msg.Blocks = append(msg.Blocks,
			Block{
				Type: contextString,
				Elements: []Text{
					{
						Type: markdown,
						Text: extractValidationError(fm.event.Message),
					},
				},
			},
		)
	}

	return msg, nil
}

func extractValidationError(msg string) string {
	errors := ""
	lines := strings.Split(msg, "\n")
	for _, line := range lines {
		if line != "" &&
			!strings.HasSuffix(line, "created") && !strings.HasSuffix(line, "created (dry run)") &&
			!strings.HasSuffix(line, "configured") && !strings.HasSuffix(line, "configured (dry run)") &&
			!strings.HasSuffix(line, "unchanged") && !strings.HasSuffix(line, "unchanged (dry run)") {
			errors += line + "\n"
		}
	}

	return errors
}

func parseRev(rev string) string {
	parts := strings.Split(rev, "/")
	if len(parts) != 2 {
		return "n/a"
	}

	return parts[1]
}

func (fm *fluxMessage) Env() string {
	return "TODO"
}

func (fm *fluxMessage) AsGithubStatus() (*githubLib.RepoStatus, error) {
	return nil, nil
}

func MessageFromFluxEvent(gitopsRepo string, event *events.Event) Message {
	return &fluxMessage{
		event:      event,
		gitopsRepo: gitopsRepo,
	}
}

func (fm *fluxMessage) RepositoryName() string {
	return ""
}

func (fm *fluxMessage) SHA() string {
	return ""
}
