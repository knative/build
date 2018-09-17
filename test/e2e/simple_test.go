// +build e2e

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

package e2e

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/knative/pkg/test/logging"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/knative/pkg/test"
)

// TestMain is called by the test binary generated by "go test", and is
// responsible for setting up and tearing down the testing environment, namely
// the test namespace.
func TestMain(m *testing.M) {
	flag.Parse()
	logging.InitializeLogger(test.Flags.LogVerbose)
	logger := logging.GetContextLogger("TestSetup")

	clients := setup(logger)
	test.CleanupOnInterrupt(func() { teardownNamespace(clients, logger) }, logger)

	code := m.Run()
	defer func() {
		// Cleanup namespace
		teardownNamespace(clients, logger)
		os.Exit(code)
	}()
}

// TestSimpleBuild tests that a simple build that does nothing interesting
// succeeds.
func TestSimpleBuild(t *testing.T) {
	logger := logging.GetContextLogger("TestSimpleBuild")
	clients := buildClients(logger)
	buildName := "simple-build"

	test.CleanupOnInterrupt(func() { teardownBuilds(clients, logger) }, logger)
	defer teardownBuilds(clients, logger)

	if _, err := clients.buildClient.builds.Create(&v1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: buildTestNamespace,
			Name:      buildName,
		},
		Spec: v1alpha1.BuildSpec{
			Timeout: "40s",
			Steps: []corev1.Container{{
				Image: "busybox",
				Args:  []string{"echo", "simple"},
			}},
		},
	}); err != nil {
		t.Fatalf("Error creating build: %v", err)
	}

	if _, err := clients.buildClient.watchBuild(buildName); err != nil {
		t.Fatalf("Error watching build: %v", err)
	}
}

// TestFailingBuild tests that a simple build that fails, fails as expected.
func TestFailingBuild(t *testing.T) {
	logger := logging.GetContextLogger("TestFailingBuild")

	clients := buildClients(logger)
	buildName := "failing-build"

	test.CleanupOnInterrupt(func() { teardownBuilds(clients, logger) }, logger)
	defer teardownBuilds(clients, logger)

	if _, err := clients.buildClient.builds.Create(&v1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: buildTestNamespace,
			Name:      buildName,
		},
		Spec: v1alpha1.BuildSpec{
			Steps: []corev1.Container{{
				Image: "busybox",
				Args:  []string{"false"}, // fails.
			}},
		},
	}); err != nil {
		t.Fatalf("Error creating build: %v", err)
	}

	if _, err := clients.buildClient.watchBuild(buildName); err == nil {
		t.Fatalf("watchBuild did not return expected error: %v", err)
	}
}

func TestBuildLowTimeout(t *testing.T) {
	logger := logging.GetContextLogger("TestBuildLowTimeout")

	clients := buildClients(logger)
	buildName := "build-low-timeout"

	test.CleanupOnInterrupt(func() { teardownBuilds(clients, logger) }, logger)
	defer teardownBuilds(clients, logger)

	buildTimeout := 50 * time.Second

	if _, err := clients.buildClient.builds.Create(&v1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: buildTestNamespace,
			Name:      buildName,
		},
		Spec: v1alpha1.BuildSpec{
			Timeout: buildTimeout.String(),
			Steps: []corev1.Container{{
				Name:    "lowtimeoutstep",
				Image:   "ubuntu",
				Command: []string{"/bin/bash"},
				Args:    []string{"-c", "sleep 2000"},
			}},
		},
	}); err != nil {
		t.Fatalf("Error creating build: %v", err)
	}

	b, err := clients.buildClient.watchBuild(buildName)
	if err == nil {
		t.Fatalf("watchBuild did not return expected BuildTimeout error")
	}

	if &b.Status == nil {
		t.Fatalf("wanted build status to be set; got nil")
	}

	successCondition := b.Status.GetCondition(v1alpha1.BuildSucceeded)

	if successCondition == nil {
		t.Fatalf("wanted build status to be set; got %q", b.Status)
	}

	// verify reason for build failure is timeout
	if successCondition.Reason != "BuildTimeout" {
		t.Fatalf("wanted BuildTimeout; got %q", successCondition.Reason)
	}
	buildDuration := b.Status.CompletionTime.Time.Sub(b.Status.StartTime.Time).Seconds()
	higherEnd := buildTimeout + 30*time.Second + 10*time.Second // build timeout + 30 sec poll time + 10 sec

	if !(buildDuration >= buildTimeout.Seconds() && buildDuration < higherEnd.Seconds()) {
		t.Fatalf("Expected the build duration to be within range %.2fs to %.2fs; but got build duration: %f, start time: %q and completed time: %q \n",
			buildTimeout.Seconds(),
			higherEnd.Seconds(),
			buildDuration,
			b.Status.StartTime.Time,
			b.Status.CompletionTime.Time,
		)
	}
}

// TestPendingBuild tests that a build with non existent node selector will remain in pending
// state until watch timeout.
func TestPendingBuild(t *testing.T) {
	logger := logging.GetContextLogger("TestPendingBuild")

	clients := buildClients(logger)
	buildName := "pending-build"

	test.CleanupOnInterrupt(func() { teardownBuilds(clients, logger) }, logger)
	defer teardownBuilds(clients, logger)

	if _, err := clients.buildClient.builds.Create(&v1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: buildTestNamespace,
			Name:      buildName,
		},
		Spec: v1alpha1.BuildSpec{
			NodeSelector: map[string]string{"disk": "fake-ssd"},
			Steps: []corev1.Container{{
				Image: "busybox",
				Args:  []string{"false"}, // fails.
			}},
		},
	}); err != nil {
		t.Fatalf("Error creating build: %v", err)
	}

	if _, err := clients.buildClient.watchBuild(buildName); err == nil {
		t.Fatalf("watchBuild did not return expected `watch timeout` error")
	}
}

// TestPodAffinity tests that a build with non existent pod affinity does not scheduled
// and fails after watch timeout
func TestPodAffinity(t *testing.T) {
	logger := logging.GetContextLogger("TestPodAffinity")

	clients := buildClients(logger)
	buildName := "affinity-build"

	test.CleanupOnInterrupt(func() { teardownBuilds(clients, logger) }, logger)
	defer teardownBuilds(clients, logger)

	if _, err := clients.buildClient.builds.Create(&v1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: buildTestNamespace,
			Name:      buildName,
		},
		Spec: v1alpha1.BuildSpec{
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					// This node affinity rule says the pod can only be placed on a node with a label whose key is kubernetes.io/e2e-az-name
					// and whose value is either e2e-az1 or e2e-az2. Test cluster does not have any nodes that meets this constraint so the build
					// will wait for pod to scheduled until timeout.
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									corev1.NodeSelectorRequirement{
										Key:      "kubernetes.io/e2e-az-name",
										Operator: corev1.NodeSelectorOpIn,
										Values:   []string{"e2e-az1", "e2e-az2"},
									}},
							},
						},
					},
				},
			},
			Steps: []corev1.Container{{
				Image: "busybox",
				Args:  []string{"true"},
			}},
		},
	}); err != nil {
		t.Fatalf("Error creating build: %v", err)
	}

	if _, err := clients.buildClient.watchBuild(buildName); err == nil {
		t.Fatalf("watchBuild did not return expected `watch timeout` error")
	}
}
