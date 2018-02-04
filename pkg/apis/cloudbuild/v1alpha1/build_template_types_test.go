/*
Copyright 2018 Google, Inc. All rights reserved.

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
)

func TestBuildTemplateConditions(t *testing.T) {
	rev := &BuildTemplate{}
	foo := &BuildTemplateCondition{
		Type:   "Foo",
		Status: "True",
	}
	bar := &BuildTemplateCondition{
		Type:   "Bar",
		Status: "True",
	}

	// Add a new condition.
	rev.Status.SetCondition(foo.Type, foo)

	if len(rev.Status.Conditions) != 1 {
		t.Fatalf("Unexpected Condition length; want 1, got %d", len(rev.Status.Conditions))
	}

	// Remove a non-existent condition.
	rev.Status.RemoveCondition(bar.Type)

	if len(rev.Status.Conditions) != 1 {
		t.Fatalf("Unexpected Condition length; want 1, got %d", len(rev.Status.Conditions))
	}

	// Add a second condition.
	rev.Status.SetCondition(bar.Type, bar)

	if len(rev.Status.Conditions) != 2 {
		t.Fatalf("Unexpected Condition length; want 2, got %d", len(rev.Status.Conditions))
	}

	// Remove an existing condition.
	rev.Status.RemoveCondition(bar.Type)

	if len(rev.Status.Conditions) != 1 {
		t.Fatalf("Unexpected Condition length; want 1, got %d", len(rev.Status.Conditions))
	}
}
