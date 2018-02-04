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

package cluster

import (
	"time"

	v1alpha1 "github.com/google/build-crd/pkg/apis/cloudbuild/v1alpha1"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	fakek8s "k8s.io/client-go/kubernetes/fake"

	buildercommon "github.com/google/build-crd/pkg/builder"
	"github.com/google/build-crd/pkg/buildtest"

	"testing"
)

const (
	namespace            = ""
	expectedErrorMessage = "stuff broke"
)

func newBuilder(cs kubernetes.Interface) *builder {
	kif := kubeinformers.NewSharedInformerFactory(cs, time.Second*30)
	return NewBuilder(cs, kif).(*builder)
}

func TestBasicFlow(t *testing.T) {
	cs := fakek8s.NewSimpleClientset()
	builder := newBuilder(cs)
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
	if bs.StartTime.IsZero() {
		t.Errorf("bs.StartTime; want non-zero, got %v", bs.StartTime)
	}
	if !bs.CompletionTime.IsZero() {
		t.Errorf("bs.CompletionTime; want zero, got %v", bs.CompletionTime)
	}
	op, err = builder.OperationFromStatus(&bs)
	if err != nil {
		t.Fatalf("Unexpected error executing OperationFromStatus: %v", err)
	}

	checksComplete := buildtest.NewWait()
	readyForUpdate := buildtest.NewWait()
	go func() {
		// Wait sufficiently long for Wait() to have been called and then
		// signal to the main test thread that it should perform the update.
		readyForUpdate.In(1 * time.Second)

		defer checksComplete.Done()
		status, err := op.Wait()
		if err != nil {
			t.Fatalf("Unexpected error waiting for builder.Operation: %v", err)
		}

		// Check that status came out how we expect.
		if !buildercommon.IsDone(status) {
			t.Errorf("IsDone(%v); wanted true, got false", status)
		}
		if status.Cluster.JobName != op.Name() {
			t.Errorf("status.Cluster.JobName; wanted %q, got %q", op.Name(), status.Cluster.JobName)
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
	}()
	// Wait until the test thread is ready for us to update things.
	readyForUpdate.Wait()

	// We should be able to fetch the Job that b.Execute() created in our fake client.
	jobsclient := cs.BatchV1().Jobs(namespace)
	job, err := jobsclient.Get(op.Name(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Unexpected error fetching Job: %v", err)
	}
	// Now modify it to look done.
	job.Status.Conditions = []batchv1.JobCondition{
		{
			Type:   batchv1.JobComplete,
			Status: corev1.ConditionTrue,
		},
	}
	job, err = jobsclient.Update(job)
	if err != nil {
		t.Fatalf("Unexpected error updating Job: %v", err)
	}

	// The informer doesn't seem to properly pick up this update via the fake,
	// so trigger the update event manually.
	builder.updateJobEvent(nil, job)

	checksComplete.WaitUntil(5*time.Second, buildtest.WaitNop, func() {
		t.Fatal("timed out in op.Wait()")
	})
}

func TestNonFinalUpdateFlow(t *testing.T) {
	cs := fakek8s.NewSimpleClientset()
	builder := newBuilder(cs)
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
	if bs.StartTime.IsZero() {
		t.Errorf("bs.StartTime; want non-zero, got %v", bs.StartTime)
	}
	if !bs.CompletionTime.IsZero() {
		t.Errorf("bs.CompletionTime; want zero, got %v", bs.CompletionTime)
	}
	op, err = builder.OperationFromStatus(&bs)
	if err != nil {
		t.Fatalf("Unexpected error executing OperationFromStatus: %v", err)
	}

	checksComplete := buildtest.NewWait()
	readyForUpdate := buildtest.NewWait()
	go func() {
		// Wait sufficiently long for Wait() to have been called and then
		// signal to the main test thread that it should perform the update.
		readyForUpdate.In(1 * time.Second)

		defer checksComplete.Done()
		status, err := op.Wait()
		if err != nil {
			t.Fatalf("Unexpected error waiting for builder.Operation: %v", err)
		}
		if status.StartTime.IsZero() {
			t.Errorf("status.StartTime; want non-zero, got %v", status.StartTime)
		}
		if status.CompletionTime.IsZero() {
			t.Errorf("status.CompletionTime; want non-zero, got %v", status.CompletionTime)
		}
	}()
	// Wait until the test thread is ready for us to update things.
	readyForUpdate.Wait()

	// We should be able to fetch the Job that b.Execute() created in our fake client.
	jobsclient := cs.BatchV1().Jobs(namespace)
	job, err := jobsclient.Get(op.Name(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Unexpected error fetching Job: %v", err)
	}
	// Make a non-terminal modification
	job.Status.Conditions = []batchv1.JobCondition{
		{
			Type:   batchv1.JobComplete,
			Status: corev1.ConditionFalse,
		},
	}
	job, err = jobsclient.Update(job)
	if err != nil {
		t.Fatalf("Unexpected error updating Job: %v", err)
	}

	// The informer doesn't seem to properly pick up this update via the fake,
	// so trigger the update event manually.
	builder.updateJobEvent(nil, job)

	// If we get a message from our Wait(), then we didn't properly ignore the
	// benign update.  If we still haven't heard anything after 5 seconds, then
	// keep going.
	checksComplete.WaitUntil(5*time.Second, func() {
		t.Fatal("Wait() returned even though our update was benign!")
	}, buildtest.WaitNop)

	// Now make it look done.
	job.Status.Conditions = []batchv1.JobCondition{
		{
			Type:   batchv1.JobComplete,
			Status: corev1.ConditionTrue,
		},
	}
	job, err = jobsclient.Update(job)
	if err != nil {
		t.Fatalf("Unexpected error updating Job: %v", err)
	}

	// The informer doesn't seem to properly pick up this update via the fake,
	// so trigger the update event manually.
	builder.updateJobEvent(nil, job)

	checksComplete.WaitUntil(5*time.Second, buildtest.WaitNop, func() {
		t.Fatal("timed out in op.Wait()")
	})
}

func TestFailureFlow(t *testing.T) {
	cs := fakek8s.NewSimpleClientset()
	builder := newBuilder(cs)
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
	if bs.StartTime.IsZero() {
		t.Errorf("bs.StartTime; want non-zero, got %v", bs.StartTime)
	}
	if !bs.CompletionTime.IsZero() {
		t.Errorf("bs.CompletionTime; want zero, got %v", bs.CompletionTime)
	}
	op, err = builder.OperationFromStatus(&bs)
	if err != nil {
		t.Fatalf("Unexpected error executing OperationFromStatus: %v", err)
	}

	checksComplete := buildtest.NewWait()
	readyForUpdate := buildtest.NewWait()
	go func() {
		// Wait sufficiently long for Wait() to have been called and then
		// signal to the main test thread that it should perform the update.
		readyForUpdate.In(1 * time.Second)

		defer checksComplete.Done()
		status, err := op.Wait()
		if err != nil {
			t.Fatalf("Unexpected error waiting for builder.Operation: %v", err)
		}

		// Check that status came out how we expect.
		if !buildercommon.IsDone(status) {
			t.Errorf("IsDone(%v); wanted true, got false", status)
		}
		if status.Cluster.JobName != op.Name() {
			t.Errorf("status.Cluster.JobName; wanted %q, got %q", op.Name(), status.Cluster.JobName)
		}
		if msg, failed := buildercommon.ErrorMessage(status); !failed || msg != expectedErrorMessage {
			t.Errorf("ErrorMessage(%v); wanted %q, got %q", status, expectedErrorMessage, msg)
		}
		if status.StartTime.IsZero() {
			t.Errorf("status.StartTime; want non-zero, got %v", status.StartTime)
		}
		if status.CompletionTime.IsZero() {
			t.Errorf("status.CompletionTime; want non-zero, got %v", status.CompletionTime)
		}
	}()
	// Wait until the test thread is ready for us to update things.
	readyForUpdate.Wait()

	// We should be able to fetch the Job that b.Execute() created in our fake client.
	jobsclient := cs.BatchV1().Jobs(namespace)
	job, err := jobsclient.Get(op.Name(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Unexpected error fetching Job: %v", err)
	}
	// Now modify it to look done.
	job.Status.Conditions = []batchv1.JobCondition{
		{
			Type:    batchv1.JobFailed,
			Status:  corev1.ConditionTrue,
			Message: expectedErrorMessage,
		},
	}
	job, err = jobsclient.Update(job)
	if err != nil {
		t.Fatalf("Unexpected error updating Job: %v", err)
	}

	// The informer doesn't seem to properly pick up this update via the fake,
	// so trigger the update event manually.
	builder.updateJobEvent(nil, job)

	checksComplete.WaitUntil(5*time.Second, buildtest.WaitNop, func() {
		t.Fatal("timed out in op.Wait()")
	})
}
