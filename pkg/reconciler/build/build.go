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
	"reflect"
	"strings"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
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

	buildsLister                listers.BuildLister
	buildTemplatesLister        listers.BuildTemplateLister
	clusterBuildTemplatesLister listers.ClusterBuildTemplateLister

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
		logger:                      logger,
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
	// TODO(jasonhall): Use a lister?
	orig, err := c.buildclientset.BuildV1alpha1().Builds(namespace).Get(name, metav1.GetOptions{})
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
	} else if err := c.updateStatus(b); err != nil {
		logger.Warn("Failed to update build status", zap.Error(err))
		return err
	}
	return err
}

func (c *Reconciler) reconcile(ctx context.Context, b *v1alpha1.Build) error {
	logger := logging.FromContext(ctx)

	// If the build doesn't define a pod name, it means it's new. We should
	// apply any template and create its pod.
	if b.Status.Cluster == nil || b.Status.Cluster.PodName == "" {
		tmpl, err := c.fetchTemplate(b)
		if err != nil {
			return err
		}
		b, err = applyTemplate(b, tmpl)
		if err != nil {
			return err
		}

		// Validate the build after applying the template.
		if err := b.Spec.Validate(); err != nil {
			return err
		}

		pod, err := resources.FromCRD(b, c.kubeclientset)
		if err != nil {
			return err
		}
		pod, err = c.kubeclientset.CoreV1().Pods(b.Namespace).Create(pod)
		if err != nil {
			return err
		}
		logger.Infof("Created pod %q for build %q", pod.Name, b.Name)
		b.Status.Cluster = &v1alpha1.ClusterSpec{
			PodName:   pod.Name,
			Namespace: b.Namespace,
		}
	}

	// TODO: Use podsLister here, for speed?
	pod, err := c.kubeclientset.CoreV1().Pods(b.Namespace).Get(b.Status.Cluster.PodName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Update the build's status from the pod's status.
	status, err := resources.StatusFromPod(pod)
	if err != nil {
		return err
	}

	// If the pod is complete and the build doesn't already have a
	// completionTIme, set the completionTime.
	status.CompletionTime = b.Status.CompletionTime
	if status.CompletionTime.IsZero() && (pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed) {
		status.CompletionTime = metav1.Now()
	}

	b.Status = *status
	return nil
}

func (c *Reconciler) updateStatus(b *v1alpha1.Build) error {
	newb, err := c.buildsLister.Builds(b.Namespace).Get(b.Name)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(newb.Status, b.Status) {
		newb.Status = b.Status
		// TODO: for CRD there's no updatestatus, so use normal update.
		_, err := c.buildclientset.BuildV1alpha1().Builds(b.Namespace).Update(newb)
		return err
	}
	return nil
}

func (c *Reconciler) fetchTemplate(b *v1alpha1.Build) (v1alpha1.BuildTemplateInterface, error) {
	if b.Spec.Template != nil {
		if b.Spec.Template.Kind == v1alpha1.ClusterBuildTemplateKind {
			tmpl, err := c.clusterBuildTemplatesLister.Get(b.Spec.Template.Name)
			// The ClusterBuildTemplate resource may not exist.
			if errors.IsNotFound(err) {
				runtime.HandleError(fmt.Errorf("cluster build template %q does not exist", b.Spec.Template.Name))
			}
			return tmpl, err
		} else {
			tmpl, err := c.buildTemplatesLister.BuildTemplates(b.Namespace).Get(b.Spec.Template.Name)
			// The BuildTemplate resource may not exist.
			if errors.IsNotFound(err) {
				runtime.HandleError(fmt.Errorf("build template %q in namespace %q does not exist", b.Spec.Template.Name, b.Namespace))
			}
			return tmpl, err
		}
	}
	return nil, nil
}

// applyTemplate applies the values in the template to the build, and replaces
// placeholders for declared parameters with the build's matching arguments.
func applyTemplate(b *v1alpha1.Build, tmpl v1alpha1.BuildTemplateInterface) (*v1alpha1.Build, error) {
	if tmpl == nil {
		return b, nil
	}
	tmpl = tmpl.Copy()
	b.Spec.Steps = tmpl.TemplateSpec().Steps
	b.Spec.Volumes = append(b.Spec.Volumes, tmpl.TemplateSpec().Volumes...)

	// Apply template arguments or parameter defaults.
	replacements := map[string]string{}
	for _, p := range tmpl.TemplateSpec().Parameters {
		if p.Default != nil {
			replacements[p.Name] = *p.Default
		}
	}
	if b.Spec.Template != nil {
		for _, a := range b.Spec.Template.Arguments {
			replacements[a.Name] = a.Value
		}
	}

	applyReplacements := func(in string) string {
		for k, v := range replacements {
			in = strings.Replace(in, fmt.Sprintf("${%s}", k), v, -1)
		}
		return in
	}

	// Apply variable expansion to steps fields.
	steps := b.Spec.Steps
	for i := range steps {
		steps[i].Name = applyReplacements(steps[i].Name)
		steps[i].Image = applyReplacements(steps[i].Image)
		for ia, a := range steps[i].Args {
			steps[i].Args[ia] = applyReplacements(a)
		}
		for ie, e := range steps[i].Env {
			steps[i].Env[ie].Value = applyReplacements(e.Value)
		}
		steps[i].WorkingDir = applyReplacements(steps[i].WorkingDir)
		for ic, c := range steps[i].Command {
			steps[i].Command[ic] = applyReplacements(c)
		}
		for iv, v := range steps[i].VolumeMounts {
			steps[i].VolumeMounts[iv].Name = applyReplacements(v.Name)
			steps[i].VolumeMounts[iv].MountPath = applyReplacements(v.MountPath)
			steps[i].VolumeMounts[iv].SubPath = applyReplacements(v.SubPath)
		}
	}

	if buildTmpl := b.Spec.Template; buildTmpl != nil && len(buildTmpl.Env) > 0 {
		// Apply variable expansion to the build's overridden
		// environment variables
		for i, e := range buildTmpl.Env {
			buildTmpl.Env[i].Value = applyReplacements(e.Value)
		}

		for i := range steps {
			steps[i].Env = applyEnvOverride(steps[i].Env, buildTmpl.Env)
		}
	}

	return b, nil
}

func applyEnvOverride(src, override []corev1.EnvVar) []corev1.EnvVar {
	result := make([]corev1.EnvVar, 0, len(src)+len(override))
	overrides := make(map[string]bool)

	for _, env := range override {
		overrides[env.Name] = true
	}

	for _, env := range src {
		if _, present := overrides[env.Name]; !present {
			result = append(result, env)
		}
	}

	return append(result, override...)
}
