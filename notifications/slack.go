package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
)

const markdown = "mrkdwn"
const section = "section"
const contextString = "context"

const githubCommitLinkFormat = "https://github.com/%s/commit/%s|%s"
const bitbucketServerLinkFormat = "http://%s/projects/%s/repos/%s/commits/%s|%s"

type slack struct {
	token          string
	defaultChannel string
	channelMapping map[string]string
}

type slackMessage struct {
	Channel string  `json:"channel"`
	Text    string  `json:"text"`
	Blocks  []Block `json:"blocks,omitempty"`
}

type Block struct {
	Type     string `json:"type"`
	Text     *Text  `json:"text,omitempty"`
	Elements []Text `json:"elements,omitempty"`
}

type Text struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *slack) send(event *GitopsEvent) error {
	slackMessage, err := s.newSlackMessage(event)
	if err != nil {
		return fmt.Errorf("cannot create slack message: %s", err)
	}

	return s.post(slackMessage)
}

func (s *slack) post(msg *slackMessage) error {
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(msg)
	if err != nil {
		logrus.Printf("Could encode message to slack: %v", err)
		return err
	}

	req, _ := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", b)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.token))
	req = req.WithContext(context.TODO())

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		logrus.Printf("could not post to slack: %v", err)
		return err
	}

	body, err := ioutil.ReadAll(res.Body)
	var parsed map[string]interface{}
	err = json.Unmarshal(body, &parsed)
	if err != nil {
		return fmt.Errorf("cannot parse slack response: %s", err)
	}
	if val, ok := parsed["ok"]; ok {
		if val != true {
			logrus.Info(string(body))
		}
	} else {
		logrus.Info(string(body))
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("could not post to slack, status: %d", res.StatusCode)
	}

	return nil
}

func (s *slack) newSlackMessage(event *GitopsEvent) (*slackMessage, error) {
	channel := s.defaultChannel
	if ch, ok := s.channelMapping[event.Manifest.Env]; ok {
		channel = ch
	}

	msg := &slackMessage{
		Channel: channel,
		Text:    "",
		Blocks:  []Block{},
	}

	if event.Status == Failure {
		msg.Text = fmt.Sprintf("Failed to roll out %s of %s", event.Manifest.App, event.Artifact.Version.RepositoryName)
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
						Text: fmt.Sprintf(":exclamation: *Error* :exclamation: \n%s", event.StatusDesc),
					},
				},
			},
		)
		msg.Blocks = append(msg.Blocks,
			Block{
				Type: contextString,
				Elements: []Text{
					{Type: markdown, Text: fmt.Sprintf(":dart: %s", strings.Title(event.Manifest.Env))},
					{Type: markdown, Text: fmt.Sprintf(":clipboard: %s", event.Artifact.Version.URL)},
				},
			},
		)
	} else {
		msg.Text = fmt.Sprintf("Rolling out %s of %s", event.Manifest.App, event.Artifact.Version.RepositoryName)
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
					{Type: markdown, Text: fmt.Sprintf(":dart: %s", strings.Title(event.Manifest.Env))},
					{Type: markdown, Text: fmt.Sprintf(":clipboard: %s", event.Artifact.Version.URL)},
					{Type: markdown, Text: fmt.Sprintf(":paperclip: %s", s.commitLink(event.GitopsRepo, event.GitopsRef))},
				},
			},
		)
	}

	return msg, nil
}

func (s *slack) commitLink(repo string, ref string) string {
	if len(ref) < 8 {
		return ""
	}
	return fmt.Sprintf(githubCommitLinkFormat, repo, ref, ref[0:7])
}
