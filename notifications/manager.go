package notifications

// GitEvent represents the git event that produced the artifact
type Event int

const (
	// Gitops event indicates a gitops write action
	Gitops Event = iota
	// Reconcile event indicates the outcome of the gitops controller reconciliation
	Reconcile
)

type Status int

const (
	Success Status = iota
	Failure
)

type GitopsEvent struct {
	Repo string
	Env  string
	App  string

	TriggerRef  string
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
	broadcast chan *GitopsEvent
}

func (m *ManagerImpl) Broadcast(event *GitopsEvent) {
}

type ManagerMock struct {
}

func (m *ManagerMock) Broadcast(event *GitopsEvent) {
}
