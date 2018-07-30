/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package fakecloudbuild implements a Fake Cloud Build API for testing.
package fakecloudbuild

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"

	cloudbuild "google.golang.org/api/cloudbuild/v1"
)

// OperationName is the name of a fake operation.
const OperationName = "projects/we-don't-care/operations/blah"

var (
	reBuildsCreate = regexp.MustCompile("^/v1/projects/[^/]+/builds")
	reOperations   = regexp.MustCompile("^/v1/projects/[^/]+/operations/[^/]+")

	// ErrorMessage is the error message to set on Operations.
	ErrorMessage = ""
)

// Closer is an interface exposing a Close method.
type Closer interface {
	// Close closes the Closer.
	Close()
}

type server struct{}

type hijackTransport struct {
	host string
}

func (t hijackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = t.host[len("http://"):] // trim off protocol.
	return http.DefaultTransport.RoundTrip(req)
}

// New returns a new fake Cloud Build API client.
func New() (*cloudbuild.Service, Closer) {
	httns := httptest.NewServer(&server{})

	cb, err := cloudbuild.New(&http.Client{
		Transport: &hijackTransport{host: httns.URL},
	})
	if err != nil {
		panic(err)
	}
	return cb, httns
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Polling operations for completion.
		if reOperations.MatchString(r.URL.Path) {
			op := cloudbuild.Operation{
				Name: r.URL.Path[4:], // Strip leading /v1/
				// TODO(mattmoor): How to wait to be done?
				Done: true,
			}
			if ErrorMessage != "" {
				op.Error = &cloudbuild.Status{Message: ErrorMessage}
			}
			_ = json.NewEncoder(w).Encode(op)
			return
		}

	case http.MethodPost:
		// Creating a build.
		if reBuildsCreate.MatchString(r.URL.Path) {
			_ = json.NewEncoder(w).Encode(cloudbuild.Operation{
				Name: OperationName,
				Done: false,
			})
			return
		}
	}

	http.Error(w, "Unexpected request path", http.StatusBadRequest)
	return
}
