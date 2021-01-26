package client

import (
	"fmt"
	"github.com/gimlet-io/gimletd/artifact"
	"testing"
)

import (
	"golang.org/x/oauth2"
)

const (
	token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"
	host  = "http://gimletd.company.com"
)

func Test_artifact(t *testing.T) {
	config := new(oauth2.Config)
	auther := config.Client(
		oauth2.NoContext,
		&oauth2.Token{
			AccessToken: token,
		},
	)

	client := NewClient(host, auther)

	artifact, err := client.ArtifactPost(&artifact.Artifact{

	})
	fmt.Println(artifact, err)
}

//func Test_artifact(t *testing.T) {
//	fixtureHandler := func(w http.ResponseWriter, r *http.Request) {
//		fmt.Fprint(w, `{
//			"pending": null,
//			"running": [
//					{
//							"id": "4696",
//							"data": "",
//							"labels": {
//									"platform": "linux/amd64",
//									"repo": "laszlocph/woodpecker"
//							},
//							"Dependencies": [],
//							"DepStatus": {},
//							"RunOn": null
//					}
//			],
//			"stats": {
//					"worker_count": 3,
//					"pending_count": 0,
//					"waiting_on_deps_count": 0,
//					"running_count": 1,
//					"completed_count": 0
//			},
//			"Paused": false
//	}`)
//	}
//
//	ts := httptest.NewServer(http.HandlerFunc(fixtureHandler))
//	defer ts.Close()
//
//	client := NewClient(ts.URL, http.DefaultClient)
//
//	info, err := client.QueueInfo()
//	if info.Stats.Workers != 3 {
//		t.Errorf("Unexpected worker count: %v, %v", info, err)
//	}
//}
