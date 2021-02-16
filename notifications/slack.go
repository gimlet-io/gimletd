package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"log"
	"net/http"
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
	logrus.Info(string(body))

	if res.StatusCode != 200 {
		log.Print("Could not post to slack, status: ", res.Status)
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
		Blocks:  nil,
	}

	if event.Status == Failure {
		msg.Text = fmt.Sprintf(
			"Rolling out %s of %s to %s, revision <%s>, failed to update gitops manifests :exclamation:",
			event.Manifest.App,
			event.Artifact.Version.RepositoryName,
			event.Manifest.Env,
			event.Artifact.Version.URL,
		)
	} else {
		msg.Text = fmt.Sprintf(
			"Rolling out %s of %s to %s, revision <%s>, gitops manifests are now updated <%s> :clipboard:",
			event.Manifest.App,
			event.Artifact.Version.RepositoryName,
			event.Manifest.Env,
			event.Artifact.Version.URL,
			s.commitLink(event.GitopsRepo, event.GitopsRef),
		)
	}

	return msg, nil
}

func (s *slack) commitLink(repo string, ref string) string {
	return fmt.Sprintf(githubCommitLinkFormat, repo, ref, ref[0:7])
}
