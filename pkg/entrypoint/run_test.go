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
	"path/filepath"
	"testing"

	"github.com/knative/pkg/logging"
)

func TestOptions_Run(t *testing.T) {
	var testCases = []struct {
		name                        string
		options                     *Options
		expectedPreRunFileContents  string
		expectedPostRunFileContents string
	}{
		{
			name: "successful_command",
			options: &Options{
				Args:                  []string{"sh", "-c", "exit 0"},
				ShouldWaitForPrevStep: true,
				PreRunFile:            "0",
				ShouldRunPostRun:      true,
				PostRunFile:           "1",
			},
			expectedPreRunFileContents:  "0",
			expectedPostRunFileContents: "0",
		},
		{
			name: "unsuccessful_command",
			options: &Options{
				Args:                  []string{"sh", "-c", "exit 1"},
				ShouldWaitForPrevStep: true,
				PreRunFile:            "0",
				ShouldRunPostRun:      true,
				PostRunFile:           "1",
			},
			expectedPreRunFileContents:  "0",
			expectedPostRunFileContents: "1",
		},
	}

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
			options := testCase.options

			// reset paths to new temp dir
			options.PreRunFile = filepath.Join(tmpDir, options.PreRunFile)
			options.PostRunFile = filepath.Join(tmpDir, options.PostRunFile)

			if options.ShouldWaitForPrevStep {
				// write the temp file it should wait for
				err := ioutil.WriteFile(options.PreRunFile, []byte("0"), os.ModePerm)
				if err != nil {
					t.Errorf("%s: error writing file to temp dir: %v", testCase.name, err)
				}
			}

			logger, _ := logging.NewLogger("", "entrypoint")
			defer logger.Sync()

			options.Run(logger)
			if options.ShouldWaitForPrevStep {
				compareFileContents(testCase.name, options.PreRunFile,
					testCase.expectedPreRunFileContents, t)
			}
			if options.ShouldRunPostRun {
				compareFileContents(testCase.name, options.PostRunFile,
					testCase.expectedPostRunFileContents, t)
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
