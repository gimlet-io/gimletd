// Copyright 2021 Laszlo Fogas
// Original structure Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"github.com/gimlet-io/gimletd/dx"
	"net/http"
	"time"
)

// Client is used to communicate with a Drone server.
type Client interface {
	// SetClient sets the http.Client.
	SetClient(*http.Client)

	// SetAddress sets the server address.
	SetAddress(string)

	// ArtifactPost creates a new artifact.
	ArtifactPost(artifact *dx.Artifact) (*dx.Artifact, error)

	// ArtifactsGet returns all artifacts in the database within the given constraints
	ArtifactsGet(
		app, branch string,
		event *dx.GitEvent,
		sourceBranch string,
		sha string,
		limit, offset int,
		since, until *time.Time,
	) ([]*dx.Artifact, error)

	// ReleasesGet returns all releases from the gitops repo within the given constraints
	ReleasesGet(
		app string,
		env string,
		limit, offset int,
		since, until *time.Time,
	) ([]*dx.Release, error)

	// ReleasesPost releases the given artifact to the given environment
	ReleasesPost(env string, artifactID string) (string, error)

	// RollbackPost rolls back to the given sha
	RollbackPost(env string, app string, targetSHA string) (string, error)

	// TrackGet returns the state of an event
	TrackGet(trackingID string) (string, string, error)
}
