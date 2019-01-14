/*
Copyright 2017 The Knative Authors.

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

package options

import (
	"flag"
	"os"
	"testing"

	"github.com/knative/build/pkg/entrypoint"
)

const (
	TestEnvVar = "TEST_ENV_VAR"
)

type TestOptions struct {
	*entrypoint.Options
}

func (o *TestOptions) ConfigVar() string {
	return TestEnvVar
}

func (o *TestOptions) AddFlags(flags *flag.FlagSet) {
	// Required to reset os.Args[1:] values used in Load()
	os.Args[1] = ""
	return
}

func (o *TestOptions) Complete(args []string) {
	return
}

func TestOptions_Load(t *testing.T) {
	tt := []struct {
		name   string
		envmap map[string]string
		in     OptionLoader
		err    error
	}{
		{"successful load", map[string]string{TestEnvVar: "hello"}, &TestOptions{}, nil},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := Load(tc.in)
			if tc.err != err {
				t.Errorf("expected err to be %v; got %v", tc.err, err)
			}
		})
	}
}
