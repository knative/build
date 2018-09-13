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

package buildtemplate

import (
	"context"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"

	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/logging/logkey"

	clientset "github.com/knative/build/pkg/client/clientset/versioned"
	buildscheme "github.com/knative/build/pkg/client/clientset/versioned/scheme"
	informers "github.com/knative/build/pkg/client/informers/externalversions/build/v1alpha1"
	listers "github.com/knative/build/pkg/client/listers/build/v1alpha1"
	cachingclientset "github.com/knative/caching/pkg/client/clientset/versioned"
	cachinginformers "github.com/knative/caching/pkg/client/informers/externalversions/caching/v1alpha1"
	cachinglisters "github.com/knative/caching/pkg/client/listers/caching/v1alpha1"
)

const controllerAgentName = "buildtemplate-controller"

// Reconciler is the controller.Reconciler implementation for BuildTemplates resources
type Reconciler struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// buildclientset is a clientset for our own API group
	buildclientset clientset.Interface
	// cachingclientset is a clientset for creating caching resources.
	cachingclientset cachingclientset.Interface

	buildTemplatesLister listers.BuildTemplateLister
	imageLister          cachinglisters.ImageLister

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
	// Add buildtemplate-controller types to the default Kubernetes Scheme so Events can be
	// logged for buildtemplate-controller types.
	buildscheme.AddToScheme(scheme.Scheme)
}

// NewController returns a new build template controller
func NewController(
	logger *zap.SugaredLogger,
	kubeclientset kubernetes.Interface,
	buildclientset clientset.Interface,
	cachingclientset cachingclientset.Interface,
	buildTemplateInformer informers.BuildTemplateInformer,
	imageInformer cachinginformers.ImageInformer,
) *controller.Impl {

	// Enrich the logs with controller name
	logger = logger.Named(controllerAgentName).With(zap.String(logkey.ControllerType, controllerAgentName))

	r := &Reconciler{
		kubeclientset:        kubeclientset,
		buildclientset:       buildclientset,
		cachingclientset:     cachingclientset,
		buildTemplatesLister: buildTemplateInformer.Lister(),
		imageLister:          imageInformer.Lister(),
		Logger:               logger,
	}
	impl := controller.NewImpl(r, logger, "BuildTemplates")

	logger.Info("Setting up event handlers")
	// Set up an event handler for when BuildTemplate resources change
	buildTemplateInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
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

	// Get the BuildTemplate resource with this namespace/name
	if _, err := c.buildTemplatesLister.BuildTemplates(namespace).Get(name); errors.IsNotFound(err) {
		// The BuildTemplate resource may no longer exist, in which case we stop processing.
		logger.Errorf("buildtemplate %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}

	// TODO: Meaningful reconciliation.
	return nil
}
