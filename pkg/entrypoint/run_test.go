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

package entrypoint

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/knative/build/pkg/entrypoint/wrapper"
	"github.com/sirupsen/logrus"
)

func TestOptions_Run(t *testing.T) {
	var testCases = []struct {
		name                          string
		args                          []string
		expectedShouldWaitForPrevStep bool
		expectedPreRunFile            string
		expectedPostRunFile           string
		expectedShouldRunPostRun      bool
	}{
		{
			name:                     "successful command",
			args:                     []string{"sh", "-c", "exit 0"},
			expectedShouldRunPostRun: true,
			expectedPostRunFile:      "0",
		},
		{
			name: "successful command with output",
			args: []string{"echo", "test"},
		},
		{
			name: "unsuccessful command",
			args: []string{"sh", "-c", "exit 12"},
		},
		{
			name: "unsuccessful command with output",
			args: []string{"sh", "-c", "echo test && exit 12"},
		},
	}

	// we write logs to the process log if wrapping fails
	// and cannot write timestamps or we can't match text
	logrus.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", testCase.name)
			if err != nil {
				t.Errorf("%s: error creating temp dir: %v", testCase.name, err)
			}
			defer func() {
				if err := os.RemoveAll(tmpDir); err != nil {
					t.Errorf("%s: error cleaning up temp dir: %v", testCase.name, err)
				}
			}()

			options := Options{
				Args: testCase.args,
				Options: &wrapper.Options{
					ShouldWaitForPrevStep: false,
					PreRunFile:            path.Join(tmpDir, "0"),
					PostRunFile:           path.Join(tmpDir, "0"),
				},
			}
			if options.ShouldWaitForPrevStep {
				compareFileContents(testCase.name, options.PreRunFile,
					testCase.expectedPreRunFile, t)
			}
			if options.ShouldRunPostRun {
				compareFileContents(testCase.name, options.PostRunFile,
					testCase.expectedPostRunFile, t)
			}
		})
	}
}

func compareFileContents(name, file, expected string, t *testing.T) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatalf("%s: could not read file: %v", name, err)
	}
	if string(data) != expected {
		t.Errorf("%s: expected contents: %q, got %q", name, expected, data)
	}
}
