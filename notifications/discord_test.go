package notifications

import (
	"testing"

	"github.com/gimlet-io/gimletd/dx"
	"github.com/gimlet-io/gimletd/model"
	"github.com/gimlet-io/gimletd/worker/events"
)

func TestSendingFluxMessage(t *testing.T) {

	msgHealthCheckPassed := fluxMessage{
		gitopsCommit: &model.GitopsCommit{
			ID:         200,
			Sha:        "76ab7d611242f7c6742f0ab662133e02b2ba2b1c",
			Status:     "Progressing",
			StatusDesc: "Health check passed",
		},
		gitopsRepo: "gimlet",
		env:        "staging",
	}

	discordMessageHealthCheckPassed, err := msgHealthCheckPassed.AsDiscordMessage()
	if err != nil {
		t.Errorf("Failed to create Discord message!")
	}

	assertEqual(t, discordMessageHealthCheckPassed.Embed.Description, ":heavy_check_mark: Applied resources from [76ab7d6](https://github.com/gimlet/commit/76ab7d611242f7c6742f0ab662133e02b2ba2b1c) are up and healthy")

	msgHealthCheckProgressing := fluxMessage{
		gitopsCommit: &model.GitopsCommit{
			ID:         200,
			Sha:        "76ab7d611242f7c6742f0ab662133e02b2ba2b1c",
			Status:     "Progressing",
			StatusDesc: "progressing",
		},
		gitopsRepo: "gimlet",
		env:        "staging",
	}

	discordMessageHealthCheckProgressing, err := msgHealthCheckProgressing.AsDiscordMessage()
	if err != nil {
		t.Errorf("Failed to create Discord message!")
	}

	assertEqual(t, discordMessageHealthCheckProgressing.Embed.Description, ":hourglass_flowing_sand: Applying gitops changes from [76ab7d6](https://github.com/gimlet/commit/76ab7d611242f7c6742f0ab662133e02b2ba2b1c)")

	msgHealthCheckFailed := fluxMessage{
		gitopsCommit: &model.GitopsCommit{
			ID:         200,
			Sha:        "76ab7d611242f7c6742f0ab662133e02b2ba2b1c",
			Status:     "ReconciliationFailed",
			StatusDesc: "progressing",
		},
		gitopsRepo: "gimlet",
		env:        "staging",
	}

	discordMessageHealthCheckFailed, err := msgHealthCheckFailed.AsDiscordMessage()
	if err != nil {
		t.Errorf("Failed to create Discord message!")
	}

	assertEqual(t, discordMessageHealthCheckFailed.Embed.Description, ":exclamation: Gitops changes from [76ab7d6](https://github.com/gimlet/commit/76ab7d611242f7c6742f0ab662133e02b2ba2b1c) failed to apply")

}

func TestSendingGitopsDeleteMessage(t *testing.T) {

	msgDeleteFailed := gitopsDeleteMessage{
		event: &events.DeleteEvent{
			Env:         "staging",
			App:         "myapp",
			TriggeredBy: "Gimlet",
			StatusDesc:  "cannot delete",
			GitopsRef:   "76ab7d611242f7c6742f0ab662133e02b2ba2b1c",
			GitopsRepo:  "testrepo",
			Status:      1,
		},
	}

	discordMessageDeleteFailed, err := msgDeleteFailed.AsDiscordMessage()
	if err != nil {
		t.Errorf("Failed to create Discord message!")
	}

	assertEqual(t, discordMessageDeleteFailed.Embed.Description, ":exclamation: *Error* :exclamation: cannot delete")

	msgPolicyDeletion := gitopsDeleteMessage{
		event: &events.DeleteEvent{
			Env:         "staging",
			App:         "myapp",
			TriggeredBy: "policy",
			StatusDesc:  "cannot delete",
			GitopsRef:   "76ab7d611242f7c6742f0ab662133e02b2ba2b1c",
			GitopsRepo:  "testrepo",
			Status:      2,
		},
	}

	discordMessagePolicyDeletion, err := msgPolicyDeletion.AsDiscordMessage()
	if err != nil {
		t.Errorf("Failed to create Discord message!")
	}

	assertEqual(t, discordMessagePolicyDeletion.Text, "Policy based deletion of myapp on staging")

}

func TestSendingGitopsDeployMessage(t *testing.T) {

	version := &dx.Version{
		RepositoryName: "testrepo",
		URL:            "https://gimlet.io",
	}

	msgSendFailure := gitopsDeployMessage{
		event: &events.DeployEvent{
			Manifest: &dx.Manifest{
				App: "myapp",
				Env: "staging",
			},
			Artifact: &dx.Artifact{
				Version: *version,
			},
			TriggeredBy: "Gimlet",
			Status:      1,
			StatusDesc:  "",
			GitopsRef:   "76ab7d611242f7c6742f0ab662133e02b2ba2b1c",
			GitopsRepo:  "testrepo",
		},
	}

	discordMessageSendFailure, err := msgSendFailure.AsDiscordMessage()
	if err != nil {
		t.Errorf("Failed to create Discord message!")
	}

	assertEqual(t, discordMessageSendFailure.Text, "Failed to roll out myapp of testrepo")

	msgSendByGimlet := gitopsDeployMessage{
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

	discordMessageSendByGimlet, err := msgSendByGimlet.AsDiscordMessage()
	if err != nil {
		t.Errorf("Failed to create Discord message!")
	}

	assertEqual(t, discordMessageSendByGimlet.Text, "Gimlet is rolling out myapp on testrepo")

}

func TestSendingGitopsRollbackMessage(t *testing.T) {

	msgRollbackFailed := gitopsRollbackMessage{
		event: &events.RollbackEvent{
			RollbackRequest: &dx.RollbackRequest{
				Env:         "staging",
				App:         "myapp",
				TargetSHA:   "76ab7d611242f7c6742f0ab662133e02b2ba2b1c",
				TriggeredBy: "Gimlet",
			},
			Status:     1,
			StatusDesc: "success",
			GitopsRefs: []string{"76ab7d611242f7c6742f0ab662133e02b2ba2b1c", "76ab7d611242f7c6742f0ab662133e02b2ba2bbb", "76ab7d611242f7c6742f0ab662133e02b2ba2lll"},
			GitopsRepo: "gimlet-io",
		},
	}

	discordMessageRollbackFailed, err := msgRollbackFailed.AsDiscordMessage()
	if err != nil {
		t.Errorf("Failed to create Discord message!")
	}

	assertEqual(t, discordMessageRollbackFailed.Text, "Failed to roll back myapp of staging")

	msgRollbackSuccess := gitopsRollbackMessage{
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

	discordMessageRollbackSuccess, err := msgRollbackSuccess.AsDiscordMessage()
	if err != nil {
		t.Errorf("Failed to create Discord message!")
	}

	assertEqual(t, discordMessageRollbackSuccess.Text, "🔙 Gimlet is rolling back myapp on staging")

}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}
