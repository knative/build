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

package cluster

import (
	"strings"
	"time"

	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	fakek8s "k8s.io/client-go/kubernetes/fake"

	buildercommon "github.com/knative/build/pkg/builder"
	"github.com/knative/build/pkg/buildtest"

	"testing"
)

const (
	namespace            = ""
	expectedErrorMessage = "stuff broke"
	expectedErrorReason  = "it was bad"
)

func newBuilder(cs kubernetes.Interface) *builder {
	kif := kubeinformers.NewSharedInformerFactory(cs, time.Second*30)
	return NewBuilder(cs, kif).(*builder)
}

func TestBasicFlow(t *testing.T) {
	cs := fakek8s.NewSimpleClientset(&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "default"}})
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
		if status.Cluster.PodName != op.Name() {
			t.Errorf("status.Cluster.PodName; wanted %q, got %q", op.Name(), status.Cluster.PodName)
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

	// We should be able to fetch the Pod that b.Execute() created in our fake client.
	podsclient := cs.CoreV1().Pods(namespace)
	pod, err := podsclient.Get(op.Name(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Unexpected error fetching Pod: %v", err)
	}
	// Now modify it to look done.
	pod.Status.Phase = corev1.PodSucceeded
	pod, err = podsclient.Update(pod)
	if err != nil {
		t.Fatalf("Unexpected error updating Pod: %v", err)
	}

	// The informer doesn't seem to properly pick up this update via the fake,
	// so trigger the update event manually.
	builder.updatePodEvent(nil, pod)

	checksComplete.WaitUntil(5*time.Second, buildtest.WaitNop, func() {
		t.Fatal("timed out in op.Wait()")
	})
}

func TestNonFinalUpdateFlow(t *testing.T) {
	cs := fakek8s.NewSimpleClientset(&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "default"}})
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

	// We should be able to fetch the Pod that b.Execute() created in our fake client.
	podsclient := cs.CoreV1().Pods(namespace)
	pod, err := podsclient.Get(op.Name(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Unexpected error fetching Pod: %v", err)
	}
	// Make a non-terminal modification
	pod.Status.Phase = corev1.PodRunning
	pod, err = podsclient.Update(pod)
	if err != nil {
		t.Fatalf("Unexpected error updating Pod: %v", err)
	}

	// The informer doesn't seem to properly pick up this update via the fake,
	// so trigger the update event manually.
	builder.updatePodEvent(nil, pod)

	// If we get a message from our Wait(), then we didn't properly ignore the
	// benign update.  If we still haven't heard anything after 5 seconds, then
	// keep going.
	checksComplete.WaitUntil(5*time.Second, func() {
		t.Fatal("Wait() returned even though our update was benign!")
	}, buildtest.WaitNop)

	// Now make it look done.
	pod.Status.Phase = corev1.PodSucceeded
	pod, err = podsclient.Update(pod)
	if err != nil {
		t.Fatalf("Unexpected error updating Pod: %v", err)
	}

	// The informer doesn't seem to properly pick up this update via the fake,
	// so trigger the update event manually.
	builder.updatePodEvent(nil, pod)

	checksComplete.WaitUntil(5*time.Second, buildtest.WaitNop, func() {
		t.Fatal("timed out in op.Wait()")
	})
}

func TestFailureFlow(t *testing.T) {
	cs := fakek8s.NewSimpleClientset(&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "default"}})
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
		if status.Cluster.PodName != op.Name() {
			t.Errorf("status.Cluster.PodName; wanted %q, got %q", op.Name(), status.Cluster.PodName)
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
		if len(status.StepStates) != 1 {
			t.Errorf("StepStates contained %d states, want 1: %+v", len(status.StepStates), status.StepStates)
		} else if status.StepStates[0].Terminated.Reason != expectedErrorReason {
			t.Errorf("StepStates[0] reason got %q, want %q", status.StepStates[0].Terminated.Reason, expectedErrorReason)
		}
	}()
	// Wait until the test thread is ready for us to update things.
	readyForUpdate.Wait()

	// We should be able to fetch the Pod that b.Execute() created in our fake client.
	podsclient := cs.CoreV1().Pods(namespace)
	pod, err := podsclient.Get(op.Name(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Unexpected error fetching Pod: %v", err)
	}
	// Now modify it to look done.
	pod.Status.Phase = corev1.PodFailed
	pod.Status.Message = expectedErrorMessage
	pod.Status.InitContainerStatuses = []corev1.ContainerStatus{{
		State: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				Reason: expectedErrorReason,
			},
		},
	}}
	pod, err = podsclient.Update(pod)
	if err != nil {
		t.Fatalf("Unexpected error updating Pod: %v", err)
	}

	// The informer doesn't seem to properly pick up this update via the fake,
	// so trigger the update event manually.
	builder.updatePodEvent(nil, pod)

	checksComplete.WaitUntil(5*time.Second, buildtest.WaitNop, func() {
		t.Fatal("timed out in op.Wait()")
	})
}

func TestStepFailureFlow(t *testing.T) {
	cs := fakek8s.NewSimpleClientset(&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "default"}})
	builder := newBuilder(cs)
	b, err := builder.BuildFromSpec(&v1alpha1.Build{
		Spec: v1alpha1.BuildSpec{
			Steps: []corev1.Container{{
				Name:    "step-name",
				Image:   "ubuntu:latest",
				Command: []string{"false"},
			}},
		},
	})
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
			t.Errorf("IsDone(%v); got false, want true", status)
		}
		if status.Cluster.PodName != op.Name() {
			t.Errorf("status.Cluster.PodName; got %q, want %q", status.Cluster.PodName, op.Name())
		}
		if msg, failed := buildercommon.ErrorMessage(status); !failed ||
			// We expect the error to contain the step name and exit code.
			!strings.Contains(msg, `"step-name"`) || !strings.Contains(msg, "128") {
			t.Errorf("ErrorMessage(%v); got %q, want %q", status, msg, expectedErrorMessage)
		}
		if status.StartTime.IsZero() {
			t.Errorf("status.StartTime; got %v, want non-zero", status.StartTime)
		}
		if status.CompletionTime.IsZero() {
			t.Errorf("status.CompletionTime; got %v, want non-zero", status.CompletionTime)
		}
	}()
	// Wait until the test thread is ready for us to update things.
	readyForUpdate.Wait()

	// We should be able to fetch the Pod that b.Execute() created in our fake client.
	podsclient := cs.CoreV1().Pods(namespace)
	pod, err := podsclient.Get(op.Name(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Unexpected error fetching Pod: %v", err)
	}
	// Now modify it to look done.
	pod.Status.Phase = corev1.PodFailed
	pod.Status.InitContainerStatuses = []corev1.ContainerStatus{{
		Name: "step-name",
		State: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode: 128,
			},
		},
		ImageID: "docker-pullable://ubuntu@sha256:deadbeef",
	}}
	pod.Status.Message = "don't expect this!"

	pod, err = podsclient.Update(pod)
	if err != nil {
		t.Fatalf("Unexpected error updating Pod: %v", err)
	}

	// The informer doesn't seem to properly pick up this update via the fake,
	// so trigger the update event manually.
	builder.updatePodEvent(nil, pod)

	checksComplete.WaitUntil(5*time.Second, buildtest.WaitNop, func() {
		t.Fatal("timed out in op.Wait()")
	})
}

func TestBasicFlowWithCredentials(t *testing.T) {
	name := "my-secret-identity"
	cs := fakek8s.NewSimpleClientset(
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
			Secrets: []corev1.ObjectReference{{
				Name: name,
			}, {
				Name: "not-annotated",
			}},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Annotations: map[string]string{
					"build.knative.dev/docker-0": "https://gcr.io",
				},
			},
			Type: corev1.SecretTypeBasicAuth,
			Data: map[string][]byte{
				corev1.BasicAuthUsernameKey: []byte("user1"),
				corev1.BasicAuthPasswordKey: []byte("password"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "not-annotated",
			},
			Type: corev1.SecretTypeBasicAuth,
			Data: map[string][]byte{
				corev1.BasicAuthUsernameKey: []byte("user2"),
				corev1.BasicAuthPasswordKey: []byte("password"),
			},
		})
	builder := newBuilder(cs)
	b, err := builder.BuildFromSpec(&v1alpha1.Build{})
	if err != nil {
		t.Fatalf("Unexpected error creating builder.Build from Spec: %v", err)
	}
	op, err := b.Execute()
	if err != nil {
		t.Fatalf("Unexpected error executing builder.Build: %v", err)
	}

	// We should be able to fetch the Pod that b.Execute() created in our fake client.
	podsclient := cs.CoreV1().Pods(namespace)
	pod, err := podsclient.Get(op.Name(), metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Unexpected error fetching Pod: %v", err)
	}

	credInit := pod.Spec.InitContainers[0]
	if got, want := len(credInit.Args), 1; got != want {
		t.Errorf("len(CredInit.Args); got %v, want %v", got, want)
	}
	if !strings.Contains(credInit.Args[0], name) {
		t.Errorf("arg[0]; got: %v, wanted string containing %q", credInit.Args[0], name)
	}
}
