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
	"fmt"
	"sync"

	"github.com/golang/glog"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	v1alpha1 "github.com/google/build-crd/pkg/apis/cloudbuild/v1alpha1"
	buildercommon "github.com/google/build-crd/pkg/builder"
	"github.com/google/build-crd/pkg/builder/cluster/convert"
)

type operation struct {
	builder   *builder
	namespace string
	name      string
	startTime metav1.Time
}

func (op *operation) Name() string {
	return op.name
}

func (op *operation) Checkpoint(status *v1alpha1.BuildStatus) error {
	status.Builder = v1alpha1.ClusterBuildProvider
	if status.Cluster == nil {
		status.Cluster = &v1alpha1.ClusterSpec{}
	}
	status.Cluster.Namespace = op.namespace
	status.Cluster.JobName = op.Name()
	status.StartTime = op.startTime
	status.SetCondition(v1alpha1.BuildComplete, &v1alpha1.BuildCondition{
		Type:               v1alpha1.BuildComplete,
		Status:             corev1.ConditionFalse,
		Reason:             "Building",
		LastTransitionTime: metav1.Now(),
	})
	return nil
}

func (op *operation) Wait() (*v1alpha1.BuildStatus, error) {
	errorCh := make(chan string)
	defer close(errorCh)

	// Ask the builder's watch loop to send a message on our channel when it sees our Job complete.
	if err := op.builder.registerDoneCallback(op.namespace, op.name, errorCh); err != nil {
		return nil, err
	}

	glog.Infof("Waiting for %q", op.Name())
	// This gets an empty string, when no error was found.
	msg := <-errorCh

	bs := &v1alpha1.BuildStatus{
		Builder: v1alpha1.ClusterBuildProvider,
		Cluster: &v1alpha1.ClusterSpec{
			Namespace: op.namespace,
			JobName:   op.Name(),
		},
		StartTime:      op.startTime,
		CompletionTime: metav1.Now(),
	}
	if msg != "" {
		bs.RemoveCondition(v1alpha1.BuildComplete)
		bs.SetCondition(v1alpha1.BuildFailed, &v1alpha1.BuildCondition{
			Type:               v1alpha1.BuildFailed,
			Status:             corev1.ConditionTrue,
			Message:            msg,
			LastTransitionTime: metav1.Now(),
		})
	} else {
		bs.SetCondition(v1alpha1.BuildComplete, &v1alpha1.BuildCondition{
			Type:               v1alpha1.BuildComplete,
			Status:             corev1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
		})
	}
	return bs, nil
}

type build struct {
	builder *builder
	body    *batchv1.Job
}

func (b *build) Execute() (buildercommon.Operation, error) {
	job, err := b.builder.kubeclient.BatchV1().Jobs(b.body.Namespace).Create(b.body)
	if err != nil {
		return nil, err
	}
	return &operation{
		builder:   b.builder,
		namespace: job.Namespace,
		name:      job.Name,
		startTime: metav1.Now(),
	}, nil
}

// NewBuilder constructs an on-cluster builder.Interface for executing Build custom resources.
func NewBuilder(kubeclient kubernetes.Interface, kubeinformers kubeinformers.SharedInformerFactory) buildercommon.Interface {
	b := &builder{
		kubeclient: kubeclient,
		callbacks:  make(map[string]chan string),
	}

	jobInformer := kubeinformers.Batch().V1().Jobs()
	jobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    b.addJobEvent,
		UpdateFunc: b.updateJobEvent,
		DeleteFunc: b.deleteJobEvent,
	})

	return b
}

type builder struct {
	kubeclient kubernetes.Interface

	// mux guards modifications to callbacks
	mux sync.Mutex
	// callbacks is keyed by Job names and stores the channel on which to
	// send a completion notification when we see that Job complete.
	// On success, an empty string is sent.
	// On failure, the Message of the failure JobCondition is sent.
	callbacks map[string]chan string
}

func (b *builder) Builder() v1alpha1.BuildProvider {
	return v1alpha1.ClusterBuildProvider
}

func (b *builder) Validate(u *v1alpha1.Build, tmpl *v1alpha1.BuildTemplate) error {
	if err := buildercommon.ValidateBuild(u, tmpl); err != nil {
		return err
	}
	if _, err := convert.FromCRD(u); err != nil {
		return err
	}
	return nil
}

func (b *builder) BuildFromSpec(u *v1alpha1.Build) (buildercommon.Build, error) {
	bld, err := convert.FromCRD(u)
	if err != nil {
		return nil, err
	}
	return &build{
		builder: b,
		body:    bld,
	}, nil
}

func (b *builder) OperationFromStatus(status *v1alpha1.BuildStatus) (buildercommon.Operation, error) {
	if status.Builder != v1alpha1.ClusterBuildProvider {
		return nil, fmt.Errorf("not a 'Cluster' builder: %v", status.Builder)
	}
	if status.Cluster == nil {
		return nil, fmt.Errorf("status.cluster cannot be empty: %v", status)
	}
	return &operation{
		builder:   b,
		namespace: status.Cluster.Namespace,
		name:      status.Cluster.JobName,
		startTime: status.StartTime,
	}, nil
}

func getKey(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

// registerDoneCallback directs the builders to send a completion notification on errorCh
// when the named Job completes.  An empty message is sent on successful completion.
func (b *builder) registerDoneCallback(namespace, name string, errorCh chan string) error {
	b.mux.Lock()
	defer b.mux.Unlock()
	if _, ok := b.callbacks[getKey(namespace, name)]; ok {
		return fmt.Errorf("another process is already waiting on %v", getKey(namespace, name))
	}
	b.callbacks[getKey(namespace, name)] = errorCh
	return nil
}

// addJobEvent handles the informer's AddFunc event for Jobs.
func (b *builder) addJobEvent(obj interface{}) {
	job := obj.(*batchv1.Job)
	ownerRef := metav1.GetControllerOf(job)

	// If this object is not owned by a Build, we should not do anything more with it.
	if ownerRef == nil || ownerRef.Kind != "Build" {
		return
	}

	// We only take action on jobs that have completed, in some way.
	msg, ok := isDone(job)
	if !ok {
		return
	}

	// Once we have a complete Job to act on, take the lock and see if anyone's watching.
	b.mux.Lock()
	defer b.mux.Unlock()
	key := getKey(job.Namespace, job.Name)
	if ch, ok := b.callbacks[key]; ok {
		// Send the person listening the message and remove this callback from our map.
		ch <- msg
		delete(b.callbacks, key)
	} else {
		glog.Errorf("Saw %q complete, but nothing was watching for it!", key)
	}
}

// updateJobEvent handles the informer's UpdateFunc event for Jobs.
func (b *builder) updateJobEvent(old, new interface{}) {
	// Same as addJobEvent(new)
	b.addJobEvent(new)
}

// deleteJobEvent handles the informer's DeleteFunc event for Jobs.
func (b *builder) deleteJobEvent(obj interface{}) {
	// TODO(mattmoor): If a job gets deleted and someone's watching, we should propagate our
	// own error message so that we don't leak a go routine waiting forever.
	glog.Errorf("NYI: delete event for: %v", obj)
}

func isDone(job *batchv1.Job) (string, bool) {
	for _, cond := range job.Status.Conditions {
		if cond.Status != corev1.ConditionTrue {
			continue
		}
		switch cond.Type {
		case batchv1.JobComplete:
			return "", true
		case batchv1.JobFailed:
			return cond.Message, true
		}
	}
	return "", false
}
