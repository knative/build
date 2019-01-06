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

package entrypoint

import (
	"encoding/json"
	"errors"
	"flag"

	"github.com/knative/build/pkg/entrypoint/wrapper"
)

// NewOptions returns an empty Options with no nil fields
func NewOptions() *Options {
	return &Options{
		Options: &wrapper.Options{},
	}
}

// Options exposes the configuration necessary
// for defining the process being watched and
// where in the image repository an upload will land.
type Options struct {
	// Args is the process and args to run
	Args []string `json:"args"`

	*wrapper.Options
}

// Validate ensures that the set of options are
// self-consistent and valid
func (o *Options) Validate() error {
	if len(o.Args) == 0 {
		return errors.New("no process to wrap specified")
	}

	return o.Options.Validate()
}

const (
	// JSONConfigEnvVar is the environment variable that
	// utilities expect to find a full JSON configuration
	// in when run.
	JSONConfigEnvVar = "ENTRYPOINT_OPTIONS"
)

// ConfigVar exposes the environment variable used
// to store serialized configuration
func (o *Options) ConfigVar() string {
	return JSONConfigEnvVar
}

// LoadConfig loads options from serialized config
func (o *Options) LoadConfig(config string) error {
	return json.Unmarshal([]byte(config), o)
}

// AddFlags binds flags to options
func (o *Options) AddFlags(flags *flag.FlagSet) {
	flags.BoolVar(&o.ShouldWaitForPrevStep, "should-wait-for-prev-step",
		DefaultShouldWaitForPrevStep, "If we should wait for prev step.")
	flags.BoolVar(&o.ShouldRunPostRun, "should-run-post-run",
		DefaultShouldRunPostRun, "If the post run step should be run after execution finishes.")
	flags.StringVar(&o.PreRunFile, "prerun-file",
		DefaultPreRunFile, "The path of the file that acts as a lock for the entrypoint.  The entrypoint binary will wait until that file is present to launch the specified command.")
	flags.StringVar(&o.PostRunFile, "postrun-file",
		DefaultPostRunFile, "The path of the file that will be written once the command finishes for the entrypoint.  This can act as a lock for other entrypoint rewritten containers.")
	o.Options.AddFlags(flags)
}

// Complete internalizes command line arguments
func (o *Options) Complete(args []string) {
	o.Args = args
}

// Encode will encode the set of options in the format that
// is expected for the configuration environment variable
func Encode(options Options) (string, error) {
	encoded, err := json.Marshal(options)
	return string(encoded), err
}
