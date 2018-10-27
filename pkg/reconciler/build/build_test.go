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

package build

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/knative/pkg/controller"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	kuberrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubeinformers "k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	clientgotesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"

	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/knative/build/pkg/builder"
	"github.com/knative/build/pkg/builder/nop"
	"github.com/knative/build/pkg/client/clientset/versioned/fake"
	informers "github.com/knative/build/pkg/client/informers/externalversions"
)

const (
	noErrorMessage = ""
)

const (
	noResyncPeriod time.Duration = 0
)

type fixture struct {
	t *testing.T

	client     *fake.Clientset
	kubeclient *k8sfake.Clientset
	objects    []runtime.Object
}

func newBuild(name string) *v1alpha1.Build {
	return &v1alpha1.Build{
		TypeMeta: metav1.TypeMeta{APIVersion: v1alpha1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
		},
		Spec: v1alpha1.BuildSpec{
			Timeout: &metav1.Duration{Duration: 20 * time.Minute},
		},
	}
}

func (f *fixture) createServceAccount() {
	f.t.Helper()

	if _, err := f.kubeclient.CoreV1().ServiceAccounts(metav1.NamespaceDefault).Create(
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "default"},
		},
	); err != nil {
		f.t.Fatalf("Failed to create ServiceAccount: %v", err)
	}
}

func (f *fixture) newController(b builder.Interface) (*controller.Impl, informers.SharedInformerFactory, kubeinformers.SharedInformerFactory) {
	k8sI := kubeinformers.NewSharedInformerFactory(f.kubeclient, noResyncPeriod)
	logger := zap.NewExample().Sugar()
	i := informers.NewSharedInformerFactory(f.client, noResyncPeriod)
	buildInformer := i.Build().V1alpha1().Builds()
	buildTemplateInformer := i.Build().V1alpha1().BuildTemplates()
	clusterBuildTemplateInformer := i.Build().V1alpha1().ClusterBuildTemplates()
	c := NewController(logger, f.kubeclient, f.client, buildInformer, buildTemplateInformer, clusterBuildTemplateInformer, b)
	return c, i, k8sI
}

func (f *fixture) updateIndex(i informers.SharedInformerFactory, bl []*v1alpha1.Build) {
	for _, f := range bl {
		i.Build().V1alpha1().Builds().Informer().GetIndexer().Add(f)
	}
}

func getKey(build *v1alpha1.Build, t *testing.T) string {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(build)
	if err != nil {
		t.Errorf("unexpected error getting key for build %v: %v", build.Name, err)
		return ""
	}
	return key
}

func TestBuildNotFoundFlow(t *testing.T) {
	bldr := &nop.Builder{}

	build := newBuild("test")
	f := &fixture{
		t:          t,
		objects:    []runtime.Object{build},
		client:     fake.NewSimpleClientset(build),
		kubeclient: k8sfake.NewSimpleClientset(),
	}

	f.createServceAccount()

	// induce failure when fetching build information in controller
	reactor := func(action clientgotesting.Action) (handled bool, ret runtime.Object, err error) {
		if action.GetVerb() == "get" && action.GetResource().Resource == "builds" {
			return true, nil, fmt.Errorf("Inducing failure for %q action of %q", action.GetVerb(), action.GetResource().Resource)
		}
		return false, nil, nil
	}
	f.client.PrependReactor("*", "*", reactor)

	stopCh := make(chan struct{})
	defer close(stopCh)

	c, i, k8sI := f.newController(bldr)
	f.updateIndex(i, []*v1alpha1.Build{build})
	i.Start(stopCh)
	k8sI.Start(stopCh)

	if err := c.Reconciler.Reconcile(context.Background(), getKey(build, t)); err == nil {
		t.Errorf("Expect error syncing build")
	}
}

func TestBuildWithBadKey(t *testing.T) {
	bldr := &nop.Builder{}

	f := &fixture{
		t:          t,
		kubeclient: k8sfake.NewSimpleClientset(),
	}
	f.createServceAccount()

	c, _, _ := f.newController(bldr)

	if err := c.Reconciler.Reconcile(context.Background(), "bad/worse/worst"); err != nil {
		t.Errorf("Unexpected error while syncing build: %s", err.Error())
	}
}

func TestBuildNotFoundError(t *testing.T) {
	bldr := &nop.Builder{}

	build := newBuild("test")
	f := &fixture{
		t:          t,
		objects:    []runtime.Object{build},
		client:     fake.NewSimpleClientset(build),
		kubeclient: k8sfake.NewSimpleClientset(),
	}
	f.createServceAccount()

	stopCh := make(chan struct{})
	defer close(stopCh)

	c, i, k8sI := f.newController(bldr)
	// Don't update build informers with test build object
	i.Start(stopCh)
	k8sI.Start(stopCh)

	if err := c.Reconciler.Reconcile(context.Background(), getKey(build, t)); err != nil {
		t.Errorf("Unexpected error while syncing build: %s", err.Error())
	}
}

func TestBuildWithNonExistentTemplates(t *testing.T) {
	for _, kind := range []v1alpha1.TemplateKind{v1alpha1.BuildTemplateKind, v1alpha1.ClusterBuildTemplateKind} {
		build := newBuild("test-buildtemplate")

		build.Spec = v1alpha1.BuildSpec{
			Template: &v1alpha1.TemplateInstantiationSpec{
				Kind: kind,
				Name: "not-existent-template",
			},
		}
		f := &fixture{
			t:          t,
			objects:    []runtime.Object{build},
			client:     fake.NewSimpleClientset(build),
			kubeclient: k8sfake.NewSimpleClientset(),
		}
		f.createServceAccount()

		stopCh := make(chan struct{})
		defer close(stopCh)

		c, i, k8sI := f.newController(&nop.Builder{})
		f.updateIndex(i, []*v1alpha1.Build{build})
		i.Start(stopCh)
		k8sI.Start(stopCh)

		if err := c.Reconciler.Reconcile(context.Background(), getKey(build, t)); err == nil {
			t.Errorf("Expect error syncing build")
		} else if !kuberrors.IsNotFound(err) {
			t.Errorf("Expect error to be not found err: %s", err.Error())
		}
	}
}
func TestBuildWithTemplate(t *testing.T) {
	tmpl := &v1alpha1.BuildTemplate{
		TypeMeta: metav1.TypeMeta{APIVersion: v1alpha1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-template",
			Namespace: metav1.NamespaceDefault,
		},
	}
	buildTemplateSpec := &v1alpha1.TemplateInstantiationSpec{
		Kind: v1alpha1.BuildTemplateKind,
		Name: tmpl.Name,
		Env:  []corev1.EnvVar{corev1.EnvVar{Value: "testvalue", Name: "testkey"}},
	}

	build := newBuild("test-buildtemplate")
	build.Spec = v1alpha1.BuildSpec{
		Template: buildTemplateSpec,
	}

	f := &fixture{
		t:          t,
		objects:    []runtime.Object{build, tmpl},
		client:     fake.NewSimpleClientset(build, tmpl),
		kubeclient: k8sfake.NewSimpleClientset(),
	}
	f.createServceAccount()

	stopCh := make(chan struct{})
	defer close(stopCh)

	c, i, k8sI := f.newController(&nop.Builder{})

	err := i.Build().V1alpha1().BuildTemplates().Informer().GetIndexer().Add(tmpl)
	if err != nil {
		t.Errorf("Unexpected error when adding cluster build template to build informer: %s", err.Error())
	}

	f.updateIndex(i, []*v1alpha1.Build{build})
	i.Start(stopCh)
	k8sI.Start(stopCh)

	if err = c.Reconciler.Reconcile(context.Background(), getKey(build, t)); err != nil {
		t.Errorf("unexpected expecting error while syncing build: %s", err.Error())
	}

	buildClient := f.client.BuildV1alpha1().Builds(build.Namespace)
	b, err := buildClient.Get(build.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("error fetching build: %v", err)
	}
	if d := cmp.Diff(b.Spec.Template, buildTemplateSpec); d != "" {
		t.Errorf("error matching build template spec: expected %#v; got %#v; diff %v", buildTemplateSpec, b.Spec.Template, d)
	}
}
func TestBasicFlows(t *testing.T) {
	tests := []struct {
		bldr                 builder.Interface
		setup                func()
		expectedErrorMessage string
	}{{
		bldr:                 &nop.Builder{},
		expectedErrorMessage: noErrorMessage,
	}, {
		bldr:                 &nop.Builder{ErrorMessage: "boom"},
		expectedErrorMessage: "boom",
	}}

	for idx, test := range tests {
		build := newBuild("test")
		f := &fixture{
			t:          t,
			objects:    []runtime.Object{build},
			client:     fake.NewSimpleClientset(build),
			kubeclient: k8sfake.NewSimpleClientset(),
		}
		f.createServceAccount()

		stopCh := make(chan struct{})
		defer close(stopCh)

		c, i, k8sI := f.newController(test.bldr)
		f.updateIndex(i, []*v1alpha1.Build{build})
		i.Start(stopCh)
		k8sI.Start(stopCh)

		// Run a single iteration of the syncHandler.
		ctx := context.Background()
		if err := c.Reconciler.Reconcile(ctx, getKey(build, t)); err != nil {
			t.Errorf("error syncing build: %v", err)
		}

		buildClient := f.client.BuildV1alpha1().Builds(build.Namespace)
		first, err := buildClient.Get(build.Name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("error fetching build: %v", err)
		}
		// Update status to current time
		first.Status.StartTime = metav1.Now()

		if builder.IsDone(&first.Status) {
			t.Errorf("First IsDone(%d); wanted not done, got done.", idx)
		}
		if msg, failed := builder.ErrorMessage(&first.Status); failed {
			t.Errorf("First ErrorMessage(%d); wanted not failed, got %q.", idx, msg)
		}

		// We have to manually update the index, or the controller won't see the update.
		f.updateIndex(i, []*v1alpha1.Build{first})

		// Run a second iteration of the syncHandler.
		if err := c.Reconciler.Reconcile(ctx, getKey(build, t)); err != nil {
			t.Errorf("error syncing build: %v", err)
		}
		// A second reconciliation will trigger an asynchronous "Wait()", which
		// should immediately return and trigger an update.  Sleep to ensure that
		// is all done before further checks.
		time.Sleep(1 * time.Second)

		second, err := buildClient.Get(build.Name, metav1.GetOptions{})
		if err != nil {
			t.Errorf("error fetching build: %v", err)
		}

		if !builder.IsDone(&second.Status) {
			t.Errorf("Second IsDone(%d, %v); wanted done, got not done.", idx, second.Status)
		}
		if msg, _ := builder.ErrorMessage(&second.Status); test.expectedErrorMessage != msg {
			t.Errorf("Second ErrorMessage(%d); wanted %q, got %q.", idx, test.expectedErrorMessage, msg)
		}
	}
}

func TestErrFlows(t *testing.T) {
	bldrErr := errors.New("not okay")
	bldr := &nop.Builder{Err: bldrErr}

	build := newBuild("test-err")
	f := &fixture{
		t:          t,
		objects:    []runtime.Object{build},
		client:     fake.NewSimpleClientset(build),
		kubeclient: k8sfake.NewSimpleClientset(),
	}
	f.createServceAccount()

	stopCh := make(chan struct{})
	defer close(stopCh)

	c, i, k8sI := f.newController(bldr)
	f.updateIndex(i, []*v1alpha1.Build{build})
	i.Start(stopCh)
	k8sI.Start(stopCh)

	if err := c.Reconciler.Reconcile(context.Background(), getKey(build, t)); err == nil {
		t.Errorf("Expect error syncing build")
	}

	// Fetch the build object and check the status
	buildClient := f.client.BuildV1alpha1().Builds(build.Namespace)
	b, err := buildClient.Get(build.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("error fetching build: %v", err)
	}

	if !builder.IsDone(&b.Status) {
		t.Error("Builder IsDone(); wanted done, got not done.")
	}
	if msg, _ := builder.ErrorMessage(&b.Status); bldrErr.Error() != msg {
		t.Errorf("Builder ErrorMessage(); wanted %q, got %q.", bldrErr.Error(), msg)
	}
}

func TestTimeoutFlows(t *testing.T) {
	build := newBuild("test")
	buffer := 1 * time.Minute

	build.Spec.Timeout = &metav1.Duration{Duration: 1 * time.Second}

	f := &fixture{
		t:          t,
		objects:    []runtime.Object{build},
		client:     fake.NewSimpleClientset(build),
		kubeclient: k8sfake.NewSimpleClientset(),
	}
	f.createServceAccount()

	stopCh := make(chan struct{})
	defer close(stopCh)

	c, i, k8sI := f.newController(&nop.Builder{})

	f.updateIndex(i, []*v1alpha1.Build{build})
	i.Start(stopCh)
	k8sI.Start(stopCh)

	ctx := context.Background()
	if err := c.Reconciler.Reconcile(ctx, getKey(build, t)); err != nil {
		t.Errorf("Not Expect error when syncing build")
	}

	buildClient := f.client.BuildV1alpha1().Builds(build.Namespace)
	first, err := buildClient.Get(build.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("error fetching build: %v", err)
	}

	// Update status to past time by substracting buffer time
	first.Status.StartTime.Time = metav1.Now().Time.Add(-buffer)

	if builder.IsDone(&first.Status) {
		t.Error("First IsDone; wanted not done, got done.")
	}

	// We have to manually update the index, or the controller won't see the update.
	f.updateIndex(i, []*v1alpha1.Build{first})

	// Run a second iteration of the syncHandler.
	if err := c.Reconciler.Reconcile(ctx, getKey(build, t)); err != nil {
		t.Errorf("Unexpected error while syncing build: %v", err)
	}

	second, err := buildClient.Get(build.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("error fetching build: %v", err)
	}

	// Ignore last transition time for comparing status objects
	var ignoreLastTransitionTime = cmpopts.IgnoreTypes(duckv1alpha1.Condition{}.LastTransitionTime.Inner.Time)

	buildStatusMsg := fmt.Sprintf("Build %q failed to finish within \"1s\"", second.Name)

	buildStatus := second.Status.GetCondition(duckv1alpha1.ConditionSucceeded)
	expectedStatus := &duckv1alpha1.Condition{
		Type:    duckv1alpha1.ConditionSucceeded,
		Status:  corev1.ConditionFalse,
		Reason:  "BuildTimeout",
		Message: buildStatusMsg,
	}

	if d := cmp.Diff(buildStatus, expectedStatus, ignoreLastTransitionTime); d != "" {
		t.Errorf("Mismatch of build status: expected %#v ; got %#v; diff %s", expectedStatus, buildStatus, d)
	}
}

func TestTimeoutFlowWithFailedOperation(t *testing.T) {
	oppErr := errors.New("test-err")
	bldr := &nop.Builder{
		OpErr: oppErr, // Include error while terminating build
	}

	build := newBuild("test")
	buffer := 10 * time.Minute

	build.Spec.Timeout = &metav1.Duration{Duration: 1 * time.Second}

	f := &fixture{
		t:          t,
		objects:    []runtime.Object{build},
		client:     fake.NewSimpleClientset(build),
		kubeclient: k8sfake.NewSimpleClientset(),
	}
	f.createServceAccount()

	stopCh := make(chan struct{})
	defer close(stopCh)

	c, i, k8sI := f.newController(bldr)

	f.updateIndex(i, []*v1alpha1.Build{build})
	i.Start(stopCh)
	k8sI.Start(stopCh)

	ctx := context.Background()
	if err := c.Reconciler.Reconcile(ctx, getKey(build, t)); err != nil {
		t.Errorf("Not Expect error when syncing build: %s", err.Error())
	}

	buildClient := f.client.BuildV1alpha1().Builds(build.Namespace)
	first, err := buildClient.Get(build.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("error fetching build: %v", err)
	}

	// Update status to past time by substracting buffer time
	first.Status.StartTime.Time = metav1.Now().Time.Add(-buffer)

	// We have to manually update the index, or the controller won't see the update.
	f.updateIndex(i, []*v1alpha1.Build{first})

	// Run a second iteration of the syncHandler to receive error from operation.
	if err = c.Reconciler.Reconcile(ctx, getKey(build, t)); err != oppErr {
		t.Errorf("Expect error %#v when syncing build", oppErr)
	}
}

func TestRunController(t *testing.T) {
	build := newBuild("test-run")

	f := &fixture{
		t:          t,
		objects:    []runtime.Object{build},
		client:     fake.NewSimpleClientset(build),
		kubeclient: k8sfake.NewSimpleClientset(),
	}

	stopCh := make(chan struct{})
	errChan := make(chan error, 1)

	defer close(errChan)

	c, i, _ := f.newController(&nop.Builder{})

	i.Start(stopCh)

	go func() {
		errChan <- c.Run(2, stopCh)
	}()

	// Shut down the controller after 2 second timeout
	go func() {
		time.Sleep(2 * time.Second)
		close(stopCh)
	}()

	buildClient := f.client.BuildV1alpha1().Builds(build.Namespace)
	b, err := buildClient.Get(build.Name, metav1.GetOptions{})
	if err != nil {
		t.Errorf("error creating build: %v", err)
	}

	// Ignore build start time when comparing
	var ignoreTime = cmpopts.IgnoreFields(v1alpha1.Build{}.Status.StartTime.Time)

	if d := cmp.Diff(b, build, ignoreTime); d != "" {
		t.Errorf("Build mismatch; diff: %s; got %v; wanted: %v", d, b, build)
	}

	if errRun := <-errChan; errRun != nil {
		t.Errorf("Unexpected error from Run(): %#v", errRun)
	}
}
