package notifications

type Message interface {
	AsSlackMessage() (*slackMessage, error)
	Env() string
}
