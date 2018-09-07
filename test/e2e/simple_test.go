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
	"log"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	kuberrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/knative/pkg/test"
)

// TestMain is called by the test binary generated by "go test", and is
// responsible for setting up and tearing down the testing environment, namely
// the test namespace.
func TestMain(m *testing.M) {
	flag.Parse()
	clients, err := newClients(test.Flags.Kubeconfig, test.Flags.Cluster, buildTestNamespace)
	if err != nil {
		log.Fatalf("newClients: %v", err)
	}

	// Ensure the test namespace exists, by trying to create it and ignoring
	// already-exists errors.
	if _, err := clients.kubeClient.Kube.CoreV1().Namespaces().Create(&corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: buildTestNamespace,
		},
	}); err == nil {
		log.Printf("Created namespace %q", buildTestNamespace)
	} else if kuberrors.IsAlreadyExists(err) {
		log.Printf("Namespace %q already exists", buildTestNamespace)
	} else {
		log.Fatalf("Error creating namespace %q: %v", buildTestNamespace, err)
	}

	defer func() {
	}()

	code := m.Run()

	// Delete the test namespace to be recreated next time.
	if err := clients.kubeClient.Kube.CoreV1().Namespaces().Delete(buildTestNamespace, &metav1.DeleteOptions{}); err != nil && !kuberrors.IsNotFound(err) {
		log.Fatalf("Error deleting namespace %q: %v", buildTestNamespace, err)
	}
	log.Printf("Deleted namespace %q", buildTestNamespace)

	os.Exit(code)
}

// TestSimpleBuild tests that a simple build that does nothing interesting
// succeeds.
func TestSimpleBuild(t *testing.T) {
	clients := setup(t)

	buildName := "simple-build"
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
	clients := setup(t)

	buildName := "failing-build"
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
	clients := setup(t)

	buildName := "build-low-timeout"
	buildTimeout := "60s"
	if _, err := clients.buildClient.builds.Create(&v1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: buildTestNamespace,
			Name:      buildName,
		},
		Spec: v1alpha1.BuildSpec{
			Timeout: buildTimeout,
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
		t.Error("watchBuild did not return expected BuildTimeout error")
	}

	successCondition := b.Status.GetCondition(v1alpha1.BuildSucceeded)

	if successCondition == nil {
		t.Fatalf("wanted build status to be set; got %q", b.Status)
	}

	// verify reason for build failure is timeout
	if successCondition.Reason != "BuildTimeout" {
		t.Fatalf("wanted BuildTimeout; got %q", b.Status.GetCondition(v1alpha1.BuildSucceeded).Reason)
	}
	buildDuration := b.Status.CompletionTime.Time.Sub(b.Status.StartTime.Time).Seconds()
	lowerEnd, _ := time.ParseDuration(buildTimeout)
	higherEnd, _ := time.ParseDuration("100s") // build timeout + 30 sec poll time + 10 sec

	if !(buildDuration > lowerEnd.Seconds() && buildDuration < higherEnd.Seconds()) {
		t.Fatalf("Expected the build duration to be within range %f to %f range; but got build start time: %q completed time: %q and duration %f \n",
			lowerEnd.Seconds(),
			higherEnd.Seconds(),
			b.Status.StartTime.Time,
			b.Status.CompletionTime.Time,
			buildDuration,
		)
	}
}
