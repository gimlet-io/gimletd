package notifications

type provider interface {
	send(msg Message) error
}
