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
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/knative/build/pkg/buildtest"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
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
	b := &Build{}
	foo := &duckv1alpha1.Condition{
		Type:   "Foo",
		Status: "True",
	}
	bar := &duckv1alpha1.Condition{
		Type:   "Bar",
		Status: "True",
	}

	// Add a new condition.
	b.Status.SetCondition(foo)

	if len(b.Status.Conditions) != 1 {
		t.Fatalf("Unexpected Condition length; want 1, got %d", len(b.Status.Conditions))
	}

	foobuildCondition := b.Status.GetConditions()[0]
	if cmp.Diff(foobuildCondition.Type, foo.Type) != "" {
		t.Fatalf("Unexpected build condition type; want %v got %v", foo.Type, foobuildCondition.Type)
	}

	if cmp.Diff(foobuildCondition.Status, foo.Status) != "" {
		t.Fatalf("Unexpected build condition status; want %v got %v", foo.Status, foobuildCondition.Type)
	}

	// Add a second condition.
	b.Status.SetCondition(bar)

	if len(b.Status.Conditions) != 2 {
		t.Fatalf("Unexpected Condition length; want 2, got %d", len(b.Status.Conditions))
	}

	barBuildCondition := b.Status.GetConditions()[0]

	if cmp.Diff(barBuildCondition.Type, bar.Type) != "" {
		t.Fatalf("Unexpected build condition type; want %v got %v", bar.Type, barBuildCondition.Type)
	}

	if cmp.Diff(barBuildCondition.Status, bar.Status) != "" {
		t.Fatalf("Unexpected build condition status; want %v got %v", bar.Status, barBuildCondition.Type)
	}

}

func TestBuildGeneration(t *testing.T) {
	b := Build{}
	if a := b.GetGeneration(); a != 0 {
		t.Errorf("empty build generation should be 0 but got: %d", a)
	}

	b.SetGeneration(5)
	if e, a := int64(5), b.GetGeneration(); e != a {
		t.Errorf("getgeneration mismatch; expected: %d got: %d", e, a)
	}
}

func TestBuildGroupVersionKind(t *testing.T) {
	b := Build{}

	epectedKind := "Build"
	if b.GetGroupVersionKind().Kind != epectedKind {
		t.Errorf("GetGroupVersionKind mismatch; expected: %v got: %v", epectedKind, b.GetGroupVersionKind().Kind)
	}
}
