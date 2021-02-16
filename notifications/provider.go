package notifications

type provider interface {
	send(event *GitopsEvent) error
}
