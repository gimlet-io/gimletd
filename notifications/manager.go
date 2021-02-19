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
	Broadcast(msg Message)
	AddProvider(provider string, token string, defaultChannel string, channelMapping string)
}

type ManagerImpl struct {
	provider  []provider
	broadcast chan Message
}

func NewManager() *ManagerImpl {
	return &ManagerImpl{
		provider:  []provider{},
		broadcast: make(chan Message),
	}
}

func (m *ManagerImpl) Broadcast(msg Message) {
	select {
	case m.broadcast <- msg:
	default:
	}
}

func (m *ManagerImpl) AddProvider(providerType string, token string, defaultChannel string, channelMapping string) {
	if providerType == "slack" {
		channelMap := map[string]string{}
		if channelMapping != "" {
			pairs := strings.Split(channelMapping, ",")
			for _, p := range pairs {
				keyValue := strings.Split(p, "=")
				channelMap[keyValue[0]] = keyValue[1]
			}
		}

		m.provider = append(m.provider,
			&slackProvider{
				token:          token,
				defaultChannel: defaultChannel,
				channelMapping: channelMap,
			},
		)
	}

	if providerType == "github" {
		m.provider = append(m.provider, newGithubProvider(token))
	}
}

func (m *ManagerImpl) Run() {
	if m.provider == nil {
		return
	}

	for {
		select {
		case message := <-m.broadcast:
			for _, p := range m.provider {
				go func() {
					err := p.send(message)
					if err != nil {
						logrus.Warnf("cannot send notification: %s ", err)
					}
				}()
			}
		}
	}
}
