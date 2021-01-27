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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gimlet-io/gimletd/artifact"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	pathArtifact = "%s/api/artifact"
	pathArtifacts = "%s/api/artifacts"
)

type client struct {
	client *http.Client
	addr   string
}

// New returns a client at the specified url.
func New(uri string) Client {
	return &client{http.DefaultClient, strings.TrimSuffix(uri, "/")}
}

// NewClient returns a client at the specified url.
func NewClient(uri string, cli *http.Client) Client {
	return &client{cli, strings.TrimSuffix(uri, "/")}
}

// SetClient sets the http.Client.
func (c *client) SetClient(client *http.Client) {
	c.client = client
}

// SetAddress sets the server address.
func (c *client) SetAddress(addr string) {
	c.addr = addr
}

// ArtifactPost creates a new user account.
func (c *client) ArtifactPost(in *artifact.Artifact) (*artifact.Artifact, error) {
	out := new(artifact.Artifact)
	uri := fmt.Sprintf(pathArtifact, c.addr)
	err := c.post(uri, in, out)
	return out, err
}

// ArtifactsGet creates a new user account.
func (c *client) ArtifactsGet() ([]*artifact.Artifact, error) {
	var out []*artifact.Artifact
	uri := fmt.Sprintf(pathArtifacts, c.addr)

	body, err := c.open(uri, "GET", nil)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}
	bodyString := string(bodyBytes)

	if bodyString == "[]" || bodyString == "{}" { // json deserializer breaks on empty arrays / objects
		return nil, nil
	}

	err = json.Unmarshal(bodyBytes, &out)
	if err != nil {
		return nil, err
	}

	if out == nil {
		return []*artifact.Artifact{}, nil
	}

	return out, err
}

func (c *client) get(rawURL string, out interface{}) error {
	return c.do(rawURL, "GET", nil, out)
}

func (c *client) post(rawURL string, in, out interface{}) error {
	return c.do(rawURL, "POST", in, out)
}

func (c *client) put(rawURL string, in, out interface{}) error {
	return c.do(rawURL, "PUT", in, out)
}

func (c *client) patch(rawURL string, in, out interface{}) error {
	return c.do(rawURL, "PATCH", in, out)
}

func (c *client) delete(rawURL string) error {
	return c.do(rawURL, "DELETE", nil, nil)
}

func (c *client) do(rawURL, method string, in, out interface{}) error {
	body, err := c.open(rawURL, method, in)
	if err != nil {
		return err
	}
	defer body.Close()

	bodyBytes, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}

	if out == nil {
		return nil
	}

	return json.Unmarshal(bodyBytes, &out)
}

func (c *client) open(rawURL, method string, in interface{}) (io.ReadCloser, error) {
	uri, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, uri.String(), nil)
	if err != nil {
		return nil, err
	}
	if in != nil {
		decoded, decodeErr := json.Marshal(in)
		if decodeErr != nil {
			return nil, decodeErr
		}
		buf := bytes.NewBuffer(decoded)
		req.Body = ioutil.NopCloser(buf)
		req.ContentLength = int64(len(decoded))
		req.Header.Set("Content-Length", strconv.Itoa(len(decoded)))
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode > http.StatusPartialContent {
		defer resp.Body.Close()
		out, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("client error %d: %s", resp.StatusCode, string(out))
	}
	return resp.Body, nil
}
