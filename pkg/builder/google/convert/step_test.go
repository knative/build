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

package convert

import (
	"testing"

	"google.golang.org/api/cloudbuild/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/knative/build/pkg/buildtest"
)

const (
	bazelYAML = "testdata/cloudbuilders/bazel/cloudbuild.yaml"
)

func TestBadSteps(t *testing.T) {
	badSteps := []cloudbuild.BuildStep{{
		// Bad Environment Variable
		// The rest of the StepSpec is omitted for brevity.
		Env: []string{"should-be-key-equals-value"},
	}}
	for _, bs := range badSteps {
		c, err := ToContainerFromStep(&bs)
		if err == nil {
			t.Errorf("ToContainerFromStep(%v); wanted error, got %v", bs, c)
		}
	}
}

func TestBadContainers(t *testing.T) {
	badContainers := []corev1.Container{{
		// Bad Environment Variable
		// The rest of the Container is omitted for brevity.
		Env: []corev1.EnvVar{{
			Name: "MY_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		}},
	}, {
		// Bad Volume
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "foo",
				MountPath: "/bar",
				ReadOnly:  true,
			},
		},
	}, {
		// Multi-part Command
		Command: []string{
			"/bin/bash",
			"-c",
			"echo Hello World",
		},
	}}
	for _, bc := range badContainers {
		s, err := ToStepFromContainer(&bc)
		if err == nil {
			t.Errorf("ToStepFromContainer(%v); wanted error, got %v", bc, s)
		}
	}
}

func TestStepRoundtripping(t *testing.T) {
	var bs cloudbuild.Build
	if err := buildtest.DataAs(bazelYAML, &bs); err != nil {
		t.Fatalf("Unexpected error in buildtest.DataAs(%q, cloudbuild.Build): %v", bazelYAML, err)
	}

	for _, step := range bs.Steps {
		c, err := ToContainerFromStep(step)
		if err != nil {
			t.Errorf("Rountripping(%v); unexpected error: %v", step, err)
		}
		result, err := ToStepFromContainer(c)
		if err != nil {
			t.Errorf("Rountripping(%v); unexpected error: %v", step, err)
		}
		// Compare the pretty json because we don't care whether slice fields are empty or nil.
		// e.g. we want omitempty semantics.
		if input, err := buildtest.PrettyJSON(step); err != nil {
			t.Errorf("Unexpected failure calling PrettyJSON(step=%v): %v", step, err)
		} else if output, err := buildtest.PrettyJSON(result); err != nil {
			t.Errorf("Unexpected failure calling PrettyJSON(result=%v): %v", result, err)
		} else if input != output {
			t.Errorf("Bad roundtrip; wanted %v, but got: %v", input, output)
		}
	}
}
