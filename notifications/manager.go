package notifications

import (
	"github.com/gimlet-io/gimletd/dx"
	"github.com/sirupsen/logrus"
	"strings"
)

type Status int

const (
	Success Status = iota
	Failure
)

type GitopsEvent struct {
	Manifest    *dx.Manifest
	Artifact    *dx.Artifact
	TriggeredBy string

	Status     Status
	StatusDesc string

	GitopsRef  string
	GitopsRepo string
}

type Manager interface {
	Broadcast(event *GitopsEvent)
}

type ManagerImpl struct {
	provider  provider
	broadcast chan *GitopsEvent
}

func NewManager(
	provider string,
	token string,
	defaultChannel string,
	channelMapping string,
) *ManagerImpl {
	if provider == "slack" {

		channelMap := map[string]string{}
		if channelMapping != "" {
			pairs := strings.Split(channelMapping, ",")
			for _, p := range pairs {
				keyValue := strings.Split(p, "=")
				channelMap[keyValue[0]] = keyValue[1]
			}
		}

		return &ManagerImpl{
			provider: &slack{
				token:          token,
				defaultChannel: defaultChannel,
				channelMapping: channelMap,
			},
			broadcast: make(chan *GitopsEvent),
		}
	}

	return &ManagerImpl{
		provider:  nil,
		broadcast: make(chan *GitopsEvent),
	}
}

func (m *ManagerImpl) Broadcast(event *GitopsEvent) {
	m.broadcast <- event
}

func (m *ManagerImpl) Run() {
	if m.provider == nil {
		return
	}

	for {
		select {
		case message := <-m.broadcast:
			go func() {
				err := m.provider.send(message)
				if err != nil {
					logrus.Warnf("cannot send notification: %s ", err)
				}
			}()
		}
	}
}
