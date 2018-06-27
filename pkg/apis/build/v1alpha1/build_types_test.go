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

package v1alpha1

import (
	"reflect"
	"testing"

	"github.com/knative/build/pkg/buildtest"
)

const bazelYAML = "testdata/cloudbuilders/bazel/cloudbuild.yaml"

func TestParsing(t *testing.T) {
	var bs BuildSpec
	if err := buildtest.DataAs(bazelYAML, &bs); err != nil {
		t.Fatalf("Unexpected error in buildtest.DataAs(%q, BuildSpec): %v", bazelYAML, err)
	}

	// Some basic checks on the body.
	if bs.Source != nil {
		t.Errorf("want no Source; got %v", bs.Source)
	}
	if len(bs.Steps) != 5 {
		t.Errorf("Wrong len(bs.Steps); wanted 5, got %d", len(bs.Steps))
	}
	for _, step := range bs.Steps {
		if len(step.Args) == 0 {
			t.Error("want len(args) != 0, got 0")
		}
	}
}

func TestBuildConditions(t *testing.T) {
	rev := &Build{}
	foo := &BuildCondition{
		Type:   "Foo",
		Status: "True",
	}
	bar := &BuildCondition{
		Type:   "Bar",
		Status: "True",
	}

	// Add a new condition.
	rev.Status.SetCondition(foo)

	if len(rev.Status.Conditions) != 1 {
		t.Fatalf("Unexpected Condition length; want 1, got %d", len(rev.Status.Conditions))
	}

	// Remove a non-existent condition.
	rev.Status.RemoveCondition(bar.Type)

	if len(rev.Status.Conditions) != 1 {
		t.Fatalf("Unexpected Condition length; want 1, got %d", len(rev.Status.Conditions))
	}

	if got, want := rev.Status.GetCondition(foo.Type), foo; !reflect.DeepEqual(got, want) {
		t.Errorf("GetCondition() = %v, want %v", got, want)
	}

	// Add a second condition.
	rev.Status.SetCondition(bar)

	if len(rev.Status.Conditions) != 2 {
		t.Fatalf("Unexpected Condition length; want 2, got %d", len(rev.Status.Conditions))
	}

	// Remove an existing condition.
	rev.Status.RemoveCondition(bar.Type)

	if len(rev.Status.Conditions) != 1 {
		t.Fatalf("Unexpected Condition length; want 1, got %d", len(rev.Status.Conditions))
	}
}
