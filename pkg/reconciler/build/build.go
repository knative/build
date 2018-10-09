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
	"reflect"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"

	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/logging/logkey"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
	clientset "github.com/knative/build/pkg/client/clientset/versioned"
	buildscheme "github.com/knative/build/pkg/client/clientset/versioned/scheme"
	informers "github.com/knative/build/pkg/client/informers/externalversions/build/v1alpha1"
	listers "github.com/knative/build/pkg/client/listers/build/v1alpha1"
	"github.com/knative/build/pkg/reconciler/build/resources"
)

const controllerAgentName = "build-controller"

// Reconciler is the controller.Reconciler implementation for Builds resources
type Reconciler struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// buildclientset is a clientset for our own API group
	buildclientset clientset.Interface

	buildsLister listers.BuildLister

	// Sugared logger is easier to use but is not as performant as the
	// raw logger. In performance critical paths, call logger.Desugar()
	// and use the returned raw logger instead. In addition to the
	// performance benefits, raw logger also preserves type-safety at
	// the expense of slightly greater verbosity.
	logger *zap.SugaredLogger
}

// Check that we implement the controller.Reconciler interface.
var _ controller.Reconciler = (*Reconciler)(nil)

func init() {
	// Add build-controller types to the default Kubernetes Scheme so Events can be
	// logged for build-controller types.
	buildscheme.AddToScheme(scheme.Scheme)
}

// NewController returns a new build template controller
func NewController(
	logger *zap.SugaredLogger,
	kubeclientset kubernetes.Interface,
	buildclientset clientset.Interface,
	buildInformer informers.BuildInformer,
) *controller.Impl {

	// Enrich the logs with controller name
	logger = logger.Named(controllerAgentName).With(zap.String(logkey.ControllerType, controllerAgentName))

	r := &Reconciler{
		kubeclientset:  kubeclientset,
		buildclientset: buildclientset,
		buildsLister:   buildInformer.Lister(),
		logger:         logger,
	}
	impl := controller.NewImpl(r, logger, "Builds")

	logger.Info("Setting up event handlers")
	// Set up an event handler for when Build resources change
	buildInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
	})

	// TODO(jasonhall): Set up a Pod informer, so that Pod updates
	// trigger Build reconciliations.

	return impl
}

// Reconcile implements controller.Reconciler
func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	logger := logging.FromContext(ctx)

	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.Errorf("invalid resource key: %s", key)
		return nil
	}

	// Get the Build resource with this namespace/name
	orig, err := c.buildsLister.Builds(namespace).Get(name)
	if errors.IsNotFound(err) {
		// The Build resource may no longer exist, in which case we stop processing.
		logger.Errorf("build %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}
	// Don't modify the informer's copy.
	b := orig.DeepCopy()

	// Reconcile this copy of the build and then write back any status
	// updates regardless of whether the reconciliation errored out.
	err = c.reconcile(ctx, b)
	if equality.Semantic.DeepEqual(orig.Status, b.Status) {
		// If we didn't change anything then don't call updateStatus.
		// This is important because the copy we loaded from the
		// informer's cache may be stale and we don't want to overwrite
		// a prior update to status with this stale state.
	} else if _, err := c.updateStatus(b); err != nil {
		logger.Warn("Failed to update build status", zap.Error(err))
		return err
	}
	return err
}

func (c *Reconciler) reconcile(ctx context.Context, b *v1alpha1.Build) error {
	logger := logging.FromContext(ctx)
	// If the build doesn't define a pod name, it doesn't have a pod. Create one.
	podName := ""
	if b.Status.Cluster == nil || b.Status.Cluster.PodName == "" {
		logger.Infof("Build %q does not have a pod, creating one", b.Name)
		b.Status.StartTime = metav1.Now()
		pod, err := resources.FromCRD(b, c.kubeclientset)
		if err != nil {
			return err
		}
		pod, err = c.kubeclientset.CoreV1().Pods(b.Namespace).Create(pod)
		if err != nil {
			return err
		}
		logger.Infof("Created pod %q for build %q", pod.Name, b.Name)
		podName = pod.Name
	} else {
		podName = b.Status.Cluster.PodName
	}
	pod, err := c.kubeclientset.CoreV1().Pods(b.Namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Update the build's status from the pod's status.
	b, err = resources.ToCRD(pod)
	if err != nil {
		return err
	}
	return nil
}

func (c *Reconciler) updateStatus(b *v1alpha1.Build) (*v1alpha1.Build, error) {
	newb, err := c.buildsLister.Builds(b.Namespace).Get(b.Name)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(newb.Status, b.Status) {
		newb.Status = b.Status
		// TODO: for CRD there's no updatestatus, so use normal update.
		return c.buildclientset.BuildV1alpha1().Builds(b.Namespace).Update(newb)
	}
	return newb, nil
}
