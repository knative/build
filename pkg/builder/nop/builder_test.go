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

package nop

import (
	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	buildercommon "github.com/knative/build/pkg/builder"

	"testing"
)

func TestBasicFlow(t *testing.T) {
	builder := Builder{}
	b, err := builder.BuildFromSpec(&v1alpha1.Build{})
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
	if bs.CreationTime.IsZero() {
		t.Errorf("bs.CreationTime; want non-zero, got %v", bs.CreationTime)
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
	if msg, failed := buildercommon.ErrorMessage(status); failed {
		t.Errorf("ErrorMessage(%v); wanted not failed, got %q", status, msg)
	}
	if status.CreationTime.IsZero() {
		t.Errorf("status.CreationTime; want non-zero, got %v", status.CreationTime)
	}
	if status.StartTime.IsZero() {
		t.Errorf("status.StartTime; want non-zero, got %v", status.StartTime)
	}
	if status.CompletionTime.IsZero() {
		t.Errorf("status.CompletionTime; want non-zero, got %v", status.CompletionTime)
	}
}

func TestBasicFlowWithError(t *testing.T) {
	expectedMsg := "Boom!"
	builder := Builder{ErrorMessage: expectedMsg}
	b, err := builder.BuildFromSpec(&v1alpha1.Build{})
	if err != nil {
		t.Fatalf("Unexpected error creating builder.Build from Spec: %v", err)
	}
	op, err := b.Execute()
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
	if msg, failed := buildercommon.ErrorMessage(status); !failed || msg != expectedMsg {
		t.Errorf("ErrorMessage(%v); wanted %q, got %q", status, expectedMsg, msg)
	}
}

func TestOperationFromStatus(t *testing.T) {
	builder := Builder{}
	op, err := builder.OperationFromStatus(&v1alpha1.BuildStatus{
		PodName: "nop-pod",
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
	if status.PodName != "nop-pod" {
		t.Errorf("PodName got %q, want %q", status.PodName, "nop-pod")
	}
	if msg, failed := buildercommon.ErrorMessage(status); failed {
		t.Errorf("ErrorMessage(%v); wanted not failed, got %q", status, msg)
	}
}
