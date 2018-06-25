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

package google

import (
	"fmt"

	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	buildercommon "github.com/knative/build/pkg/builder"

	"github.com/knative/build/pkg/builder/google/fakecloudbuild"

	"testing"
)

var (
	sampleBuild = &v1alpha1.Build{}
	project     = "frugal-function-123"
)

func newBuilder() (buildercommon.Interface, fakecloudbuild.Closer) {
	fb, c := fakecloudbuild.New()
	return NewBuilder(fb, project), c
}

func TestBasicFlow(t *testing.T) {
	builder, c := newBuilder()
	defer c.Close()
	b, err := builder.BuildFromSpec(sampleBuild)
	if err != nil {
		t.Fatalf("Unexpected error creating builder.Build from Spec: %v", err)
	}
	op, err := b.Execute()
	if err != nil {
		t.Fatalf("Unexpected error executing builder.Build: %v", err)
	}

	var bs v1alpha1.BuildStatus
	if err := op.Checkpoint(&bs); err != nil {
		t.Fatalf("Unexpected error executing op.Checkpoint: %v", err)
	}
	if buildercommon.IsDone(&bs) {
		t.Errorf("IsDone(%v); wanted not done, got done.", bs)
	}
	op, err = builder.OperationFromStatus(&bs)
	if err != nil {
		t.Fatalf("Unexpected error executing OperationFromStatus: %v", err)
	}
	if bs.StartTime.IsZero() {
		t.Errorf("bs.StartTime; want non-zero, got %v", bs.StartTime)
	}
	if !bs.CompletionTime.IsZero() {
		t.Errorf("bs.CompletionTime; want zero, got %v", bs.CompletionTime)
	}

	status, err := op.Wait()
	if err != nil {
		t.Fatalf("Unexpected error waiting for builder.Operation: %v", err)
	}

	// Check that status came out how we expect.
	if !buildercommon.IsDone(status) {
		t.Errorf("IsDone(%v); wanted true, got false", status)
	}
	if status.Google.Operation != op.Name() {
		t.Errorf("status.Google.Operation; wanted %q, got %q", op.Name(), status.Google.Operation)
	}
	if msg, failed := buildercommon.ErrorMessage(status); failed {
		t.Errorf("ErrorMessage(%v); wanted not failed, got %q", status, msg)
	}
	if status.StartTime.IsZero() {
		t.Errorf("status.StartTime; want non-zero, got %v", status.StartTime)
	}
	if status.CompletionTime.IsZero() {
		t.Errorf("status.CompletionTime; want non-zero, got %v", status.CompletionTime)
	}
}

func TestOperationFromStatus(t *testing.T) {
	builder, c := newBuilder()
	defer c.Close()
	op, err := builder.OperationFromStatus(&v1alpha1.BuildStatus{
		Builder: v1alpha1.GoogleBuildProvider,
		Google: &v1alpha1.GoogleSpec{
			Operation: fmt.Sprintf("projects/%s/operations/123", project),
		},
	})
	if err != nil {
		t.Fatalf("Unexpected error executing builder.Build: %v", err)
	}
	status, err := op.Wait()
	if err != nil {
		t.Fatalf("Unexpected error waiting for builder.Operation: %v", err)
	}

	// Check that status came out how we expect.
	if !buildercommon.IsDone(status) {
		t.Errorf("IsDone(%v); wanted true, got false", status)
	}
	if status.Google.Operation != op.Name() {
		t.Errorf("status.Google.Operation; wanted %q, got %q", op.Name(), status.Google.Operation)
	}
	if msg, failed := buildercommon.ErrorMessage(status); failed {
		t.Errorf("ErrorMessage(%v); wanted not failed, got %q", status, msg)
	}
}

func TestOperationWithFailure(t *testing.T) {
	builder, c := newBuilder()
	defer c.Close()
	op, err := builder.OperationFromStatus(&v1alpha1.BuildStatus{
		Builder: v1alpha1.GoogleBuildProvider,
		Google: &v1alpha1.GoogleSpec{
			Operation: fmt.Sprintf("projects/%s/operations/123", project),
		},
	})
	if err != nil {
		t.Fatalf("Unexpected error executing builder.Build: %v", err)
	}
	// Set the error message we expect.
	fakecloudbuild.ErrorMessage = "fail"
	status, err := op.Wait()
	if err != nil {
		t.Fatalf("Unexpected error waiting for builder.Operation: %v", err)
	}

	// Check that status came out how we expect.
	if !buildercommon.IsDone(status) {
		t.Errorf("IsDone(%v); wanted true, got false", status)
	}
	if status.Google.Operation != op.Name() {
		t.Errorf("status.Google.Operation; wanted %q, got %q", op.Name(), status.Google.Operation)
	}
	if msg, failed := buildercommon.ErrorMessage(status); !failed || msg != "fail" {
		t.Errorf("ErrorMessage(%v); wanted %q, got %q", status, "fail", msg)
	}
}
