package notifications

import (
	"fmt"
	"github.com/fluxcd/pkg/recorder"
	"strings"
	"time"
)

type fluxMessage struct {
	event *recorder.Event
}

func (fm *fluxMessage) AsSlackMessage() (*slackMessage, error) {
	msg := &slackMessage{
		Text:   "",
		Blocks: []Block{},
	}

	msg.Text = fmt.Sprintf(
		"%s: %s - %s/%s: %s - %s",
		fm.event.Timestamp.Format(time.RFC3339),
		fm.event.InvolvedObject.Kind,
		fm.event.InvolvedObject.Namespace,
		fm.event.InvolvedObject.Name,
		fm.event.Message,
		fm.event.Reason,
	)
	if fm.event.Severity == "error" {
		msg.Text = ":exclamation: :exclamation:" + msg.Text
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

	var sb strings.Builder
	for key, value := range fm.event.Metadata {
		sb.WriteString(key + ": " + value)
	}

	if len(sb.String()) > 0 {
		msg.Blocks = append(msg.Blocks,
			Block{
				Type: contextString,
				Elements: []Text{
					{
						Type: markdown,
						Text: sb.String(),
					},
				},
			},
		)
	}

	return msg, nil
}

func (fm *fluxMessage) Env() string {
	return "TODO"
}

func MessageFromFluxEvent(event *recorder.Event) Message {
	return &fluxMessage{
		event: event,
	}
}
