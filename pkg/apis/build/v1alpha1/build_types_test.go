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
	"github.com/knative/pkg/apis"
	"github.com/knative/pkg/apis/duck"
	duckv1beta1 "github.com/knative/pkg/apis/duck/v1beta1"
)

func TestBuildImplementsConditions(t *testing.T) {
	if err := duck.VerifyType(&Build{}, &duckv1beta1.Conditions{}); err != nil {
		t.Errorf("Expect Build to implement duck verify type: err %#v", err)
	}
}

func TestBuildConditions(t *testing.T) {
	b := &Build{}
	foo := &apis.Condition{
		Type:   "Foo",
		Status: "True",
	}
	bar := &apis.Condition{
		Type:   "Bar",
		Status: "True",
	}

	var ignoreVolatileTime = cmp.Comparer(func(_, _ apis.VolatileTime) bool {
		return true
	})

	// Add a new condition.
	b.Status.SetCondition(foo)

	want := apis.Conditions{*foo}
	if diff := cmp.Diff(b.Status.GetConditions(), want, ignoreVolatileTime); diff != "" {
		t.Errorf("Unexpected build condition type; %s", diff)
	}

	fooStatus := b.Status.GetCondition(foo.Type)
	if diff := cmp.Diff(fooStatus, foo, ignoreVolatileTime); diff != "" {
		t.Errorf("Unexpected build condition type; %s", diff)
	}

	// Add a second condition.
	b.Status.SetCondition(bar)

	want = apis.Conditions{*bar, *foo}

	if d := cmp.Diff(b.Status.GetConditions(), want, ignoreVolatileTime); d != "" {
		t.Fatalf("Unexpected build condition type; want %v got %v; diff %s", want, b.Status.GetConditions(), d)
	}
}

func TestBuildGroupVersionKind(t *testing.T) {
	b := Build{}

	expectedKind := "Build"
	if b.GetGroupVersionKind().Kind != expectedKind {
		t.Errorf("GetGroupVersionKind mismatch; expected: %v got: %v", expectedKind, b.GetGroupVersionKind().Kind)
	}
}
