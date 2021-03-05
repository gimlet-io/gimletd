package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
)

const markdown = "mrkdwn"
const section = "section"
const contextString = "context"

const githubCommitLinkFormat = "<https://github.com/%s/commit/%s|%s>"
const bitbucketServerLinkFormat = "<http://%s/projects/%s/repos/%s/commits/%s|%s>"

type slackProvider struct {
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

func (s *slackProvider) send(msg Message) error {
	slackMessage, err := msg.AsSlackMessage()
	if err != nil {
		return fmt.Errorf("cannot create slack message: %s", err)
	}

	if slackMessage == nil {
		return nil
	}

	channel := s.defaultChannel
	if ch, ok := s.channelMapping[msg.Env()]; ok {
		channel = ch
	}
	slackMessage.Channel = channel

	return s.post(slackMessage)
}

func (s *slackProvider) post(msg *slackMessage) error {
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

func commitLink(repo string, ref string) string {
	if len(ref) < 8 {
		return ""
	}
	return fmt.Sprintf(githubCommitLinkFormat, repo, ref, ref[0:7])
}