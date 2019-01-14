/*
Copyright 2018 The Knative Authors.

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

package wrapper

import (
	"flag"
)

// Options exposes the configuration options
// used when wrapping test execution
type Options struct {
	// ShouldWaitForPrevStep will be written with the exit code
	// of the test process or an internal error code
	// if the entrypoint fails.
	ShouldWaitForPrevStep bool `json:"shouldWaitForPrevStep"`

	// PreRunFile will be written with the exit code
	// of the test process or an internal error code
	// if the entrypoint fails.
	PreRunFile string `json:"preRunFile"`

	// ShouldWaitForPrevStep will be written with the exit code
	// of the test process or an internal error code
	// if the entrypoint fails.
	ShouldRunPostRun bool `json:"shouldRunPostRun"`

	// PostRunFile will be written with the exit code
	// of the test process or an internal error code
	// if the entrypoint fails or if it succeeds
	PostRunFile string `json:"postRunFile"`
}

// AddFlags adds flags to the FlagSet that populate
// the wrapper options struct provided.
func (o *Options) AddFlags(fs *flag.FlagSet) {
	return
}

// Validate ensures that the set of options are
// self-consistent and valid
func (o *Options) Validate() error {
	return nil
}
