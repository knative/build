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
	"fmt"
	"time"

	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	clientset "github.com/knative/build/pkg/client/clientset/versioned"
	buildscheme "github.com/knative/build/pkg/client/clientset/versioned/scheme"
	informers "github.com/knative/build/pkg/client/informers/externalversions/build/v1alpha1"
	listers "github.com/knative/build/pkg/client/listers/build/v1alpha1"
	"github.com/knative/build/pkg/reconciler"
	"github.com/knative/build/pkg/reconciler/build/resources"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/logging/logkey"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"
)

const controllerAgentName = "build-controller"

// Reconciler is the controller.Reconciler implementation for Builds resources
type Reconciler struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// buildclientset is a clientset for our own API group
	buildclientset clientset.Interface

	buildsLister                listers.BuildLister
	buildTemplatesLister        listers.BuildTemplateLister
	clusterBuildTemplatesLister listers.ClusterBuildTemplateLister

	// Sugared logger is easier to use but is not as performant as the
	// raw logger. In performance critical paths, call logger.Desugar()
	// and use the returned raw logger instead. In addition to the
	// performance benefits, raw logger also preserves type-safety at
	// the expense of slightly greater verbosity.
	Logger *zap.SugaredLogger
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
	kubeinformers kubeinformers.SharedInformerFactory,
	buildclientset clientset.Interface,
	buildInformer informers.BuildInformer,
	buildTemplateInformer informers.BuildTemplateInformer,
	clusterBuildTemplateInformer informers.ClusterBuildTemplateInformer,
) *controller.Impl {

	// Enrich the logs with controller name
	logger = logger.Named(controllerAgentName).With(zap.String(logkey.ControllerType, controllerAgentName))

	r := &Reconciler{
		kubeclientset:               kubeclientset,
		buildclientset:              buildclientset,
		buildsLister:                buildInformer.Lister(),
		buildTemplatesLister:        buildTemplateInformer.Lister(),
		clusterBuildTemplatesLister: clusterBuildTemplateInformer.Lister(),
		Logger:                      logger,
	}
	impl := controller.NewImpl(r, logger, "Builds",
		reconciler.MustNewStatsReporter("Builds", r.Logger))

	logger.Info("Setting up event handlers")
	// Set up an event handler for when Build resources change
	buildInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
	})

	// Set up a Pod informer, so that Pod updates trigger Build
	// reconciliations.
	kubeinformers.Core().V1().Pods().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    r.addPodEvent,
		UpdateFunc: r.updatePodEvent,
		DeleteFunc: r.deletePodEvent,
	})

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
	build, err := c.buildsLister.Builds(namespace).Get(name)
	if errors.IsNotFound(err) {
		// The Build resource may no longer exist, in which case we stop processing.
		logger.Errorf("build %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}

	// Don't mutate the informer's copy of our object.
	build = build.DeepCopy()

	// Validate build
	if err = c.validateBuild(build); err != nil {
		c.Logger.Errorf("Failed to validate build: %v", err)
		return err
	}

	// If the build's done, then ignore it.
	if isDone(&build.Status) {
		return nil
	}

	// If the build is ongoing, check if it's timed out.
	if build.Status.Cluster != nil && build.Status.Cluster.PodName != "" {
		// Check if build has timed out
		return c.checkTimeout(build)
	}

	// If the build hasn't started yet, create a Pod for it and record that
	// pod's name in the build status.
	p, err := c.startPodForBuild(build)
	if err != nil {
		build.Status.SetCondition(&duckv1alpha1.Condition{
			Type:    v1alpha1.BuildSucceeded,
			Status:  corev1.ConditionFalse,
			Reason:  "BuildExecuteFailed",
			Message: err.Error(),
		})
		if err := c.updateStatus(build); err != nil {
			return err
		}
		return err
	}

	// If Pod creation was successful, update the Build's status.
	bs := resources.BuildStatusFromPod(p, build.Spec)
	bs.StartTime = &metav1.Time{time.Now()}
	build.Status = bs
	return c.updateStatus(build)
}

func (c *Reconciler) updateStatus(u *v1alpha1.Build) error {
	newb, err := c.buildclientset.BuildV1alpha1().Builds(u.Namespace).Get(u.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	newb.Status = u.Status

	// Until #38113 is merged, we must use Update instead of UpdateStatus to
	// update the Status block of the Build resource. UpdateStatus will not
	// allow changes to the Spec of the resource, which is ideal for ensuring
	// nothing other than resource status has been updated.
	_, err = c.buildclientset.BuildV1alpha1().Builds(u.Namespace).Update(newb)
	return err
}

// addPodEvent handles the informer's AddFunc event for Pods.
func (c *Reconciler) addPodEvent(obj interface{}) {
	p, ok := obj.(*corev1.Pod)
	if !ok {
		return
	}
	ownerRef := metav1.GetControllerOf(p)

	// If this object is not owned by a Build, we should not do anything
	// more with it.
	if ownerRef == nil || ownerRef.Kind != "Build" {
		return
	}

	// Get the build for this pod.
	buildName := ownerRef.Name
	build, err := c.buildsLister.Builds(p.Namespace).Get(buildName)
	if err != nil {
		c.Logger.Errorf("Error getting build %q for pod %q in namespace %q", buildName, p.Name, p.Namespace)
		return
	}

	// Update the build's status from the pod's status.
	build = build.DeepCopy()
	build.Status = resources.BuildStatusFromPod(p, build.Spec)
	if err := c.updateStatus(build); err != nil {
		c.Logger.Errorf("Error updating build %q in response to pod event: %v", buildName, err)
	}
}

// updatePodEvent handles the informer's UpdateFunc event for Pods.
func (c *Reconciler) updatePodEvent(old, new interface{}) {
	c.addPodEvent(new)
}

// deletePodEvent handles the informer's DeleteFunc event for Pods.
func (c *Reconciler) deletePodEvent(obj interface{}) {
	// TODO(mattmoor): If a pod gets deleted and someone's watching, we should propagate our
	// own error message so that we don't leak a go routine waiting forever.
	c.Logger.Errorf("NYI: delete event for: %v", obj)
}

// startPodForBuild starts a new Pod to execute the build.
//
// This applies any build template that's specified, and creates the pod.
func (c *Reconciler) startPodForBuild(build *v1alpha1.Build) (*corev1.Pod, error) {
	namespace := build.Namespace
	var tmpl v1alpha1.BuildTemplateInterface
	var err error
	if build.Spec.Template != nil {
		if build.Spec.Template.Kind == v1alpha1.ClusterBuildTemplateKind {
			tmpl, err = c.clusterBuildTemplatesLister.Get(build.Spec.Template.Name)
			if err != nil {
				// The ClusterBuildTemplate resource may not exist.
				if errors.IsNotFound(err) {
					runtime.HandleError(fmt.Errorf("cluster build template %q does not exist", build.Spec.Template.Name))
				}
				return nil, err
			}
		} else {
			tmpl, err = c.buildTemplatesLister.BuildTemplates(namespace).Get(build.Spec.Template.Name)
			if err != nil {
				// The BuildTemplate resource may not exist.
				if errors.IsNotFound(err) {
					runtime.HandleError(fmt.Errorf("build template %q in namespace %q does not exist", build.Spec.Template.Name, namespace))
				}
				return nil, err
			}
		}
	}
	build, err = ApplyTemplate(build, tmpl)
	if err != nil {
		return nil, err
	}

	p, err := resources.MakePod(build, c.kubeclientset)
	if err != nil {
		return nil, err
	}
	return c.kubeclientset.CoreV1().Pods(p.Namespace).Create(p)
}

func (c *Reconciler) terminatePod(namespace, name string) error {
	if err := c.kubeclientset.CoreV1().Pods(namespace).Delete(name, &metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

// IsDone returns true if the build's status indicates the build is done.
func isDone(status *v1alpha1.BuildStatus) bool {
	cond := status.GetCondition(v1alpha1.BuildSucceeded)
	return cond != nil && cond.Status != corev1.ConditionUnknown
}

func (c *Reconciler) checkTimeout(build *v1alpha1.Build) error {
	namespace := build.Namespace
	if c.isTimeout(&build.Status, build.Spec.Timeout) {
		c.Logger.Infof("Build %q is timeout", build.Name)
		if err := c.kubeclientset.CoreV1().Pods(namespace).Delete(build.Status.Cluster.PodName, &metav1.DeleteOptions{}); err != nil && !errors.IsNotFound(err) {
			c.Logger.Errorf("Failed to terminate pod: %v", err)
			return err
		}

		timeoutMsg := fmt.Sprintf("Build %q failed to finish within %q", build.Name, build.Spec.Timeout.Duration.String())
		build.Status.SetCondition(&duckv1alpha1.Condition{
			Type:    v1alpha1.BuildSucceeded,
			Status:  corev1.ConditionFalse,
			Reason:  "BuildTimeout",
			Message: timeoutMsg,
		})
		// update build completed time
		build.Status.CompletionTime = &metav1.Time{time.Now()}

		if err := c.updateStatus(build); err != nil {
			c.Logger.Errorf("Failed to update status for pod: %v", err)
			return err
		}
	}
	return nil
}

// IsTimeout returns true if the build's execution time is greater than
// specified build spec timeout.
func (c *Reconciler) isTimeout(status *v1alpha1.BuildStatus, buildTimeout *metav1.Duration) bool {
	var timeout time.Duration
	var defaultTimeout = 10 * time.Minute

	if status == nil {
		return false
	}

	if buildTimeout == nil {
		// Set default timeout to 10 minute if build timeout is not set
		timeout = defaultTimeout
	} else {
		timeout = buildTimeout.Duration
	}

	// If build has not started timeout, startTime should be zero.
	if status.StartTime == nil {
		return false
	}
	over := time.Since(status.StartTime.Time) > timeout
	if over {
		c.Logger.Infof("Build has timed out!")
	}
	c.Logger.Infof("Build timeout=%s, runtime=%s", timeout, time.Since(status.StartTime.Time))
	return over
}
