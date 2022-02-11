package notifications

import (
	"testing"

	"github.com/gimlet-io/gimletd/dx"
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/worker/events"
)

// func TestDiscordBot(t *testing.T) {
// 	p := &DiscordProvider{
// 		Token:     "OTQxNjAzNDk5ODk5MjQ0NTc0.YgYWmA._Og1LRgO356obmm0zox3Hb4AVeo",
// 		ChannelID: "940971232847884329",
// 	}

// 	msg := fluxMessage{}
// 	err := p.send(&msg)
// 	if err != nil {
// 		t.Errorf("Sending message in Discord must not return error!")
// 	}
// }

func TestSendingFluxMessage(t *testing.T) {

	msg := fluxMessage{
		gitopsCommit: &model.GitopsCommit{
			ID:         200,
			Sha:        "76ab7d611242f7c6742f0ab662133e02b2ba2b1c",
			Status:     "Processing",
			StatusDesc: "Health check passed",
		},
		gitopsRepo: "gimlet",
		env:        "staging",
	}

	discordMessage, err := msg.AsDiscordMessage()
	if err != nil {
		t.Errorf("Failed to create Discord message!")
	}

	assertEqual(t, discordMessage.Text, "Health check")

}

func TestSendingGitopsDeleteMessage(t *testing.T) {

	msg := gitopsDeleteMessage{
		event: &events.DeleteEvent{
			Env:         "staging",
			App:         "myapp",
			TriggeredBy: "Gimlet",
			StatusDesc:  "",
			GitopsRef:   "76ab7d611242f7c6742f0ab662133e02b2ba2b1c",
			GitopsRepo:  "testrepo",
		},
	}

	discordMessage, err := msg.AsDiscordMessage()
	if err != nil {
		t.Errorf("Failed to create Discord message!")
	}

	assertEqual(t, discordMessage.Text, "Gimlet is deleting myapp on staging")

}

func TestSendingGitopsDeployMessage(t *testing.T) {

	version := &dx.Version{
		RepositoryName: "testrepo",
		URL:            "https://gimlet.io",
	}

	msg := gitopsDeployMessage{
		event: &events.DeployEvent{
			Manifest: &dx.Manifest{
				App: "myapp",
				Env: "staging",
			},
			Artifact: &dx.Artifact{
				Version: *version,
			},
			TriggeredBy: "Gimlet",
			Status:      0,
			StatusDesc:  "",
			GitopsRef:   "76ab7d611242f7c6742f0ab662133e02b2ba2b1c",
			GitopsRepo:  "testrepo",
		},
	}

	discordMessage, err := msg.AsDiscordMessage()
	if err != nil {
		t.Errorf("Failed to create Discord message!")
	}

	assertEqual(t, discordMessage.Text, "Gimlet is rolling out myapp on testrepo")

}

func TestSendingGitopsRollbackMessage(t *testing.T) {

	msg := gitopsRollbackMessage{
		event: &events.RollbackEvent{
			RollbackRequest: &dx.RollbackRequest{
				Env:         "staging",
				App:         "myapp",
				TargetSHA:   "76ab7d611242f7c6742f0ab662133e02b2ba2b1c",
				TriggeredBy: "Gimlet",
			},
			Status:     0,
			StatusDesc: "success",
			GitopsRefs: []string{"76ab7d611242f7c6742f0ab662133e02b2ba2b1c", "76ab7d611242f7c6742f0ab662133e02b2ba2bbb", "76ab7d611242f7c6742f0ab662133e02b2ba2lll"},
			GitopsRepo: "gimlet-io",
		},
	}

	discordMessage, err := msg.AsDiscordMessage()
	if err != nil {
		t.Errorf("Failed to create Discord message!")
	}

	assertEqual(t, discordMessage.Text, "🔙 Gimlet is rolling back myapp on staging")

}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}
