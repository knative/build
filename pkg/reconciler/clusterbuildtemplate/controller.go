/*
Copyright 2019 The Knative Authors

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

package clusterbuildtemplate

import (
	"context"

	"go.uber.org/zap"
	"k8s.io/client-go/tools/cache"

	buildclient "github.com/knative/build/pkg/client/injection/client"
	cbtinformer "github.com/knative/build/pkg/client/injection/informers/build/v1alpha1/clusterbuildtemplate"
	cachingclient "github.com/knative/caching/pkg/client/injection/client"
	imageinformer "github.com/knative/caching/pkg/client/injection/informers/caching/v1alpha1/image"
	"github.com/knative/pkg/injection/clients/kubeclient"

	"github.com/knative/pkg/configmap"
	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/logging/logkey"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
)

const controllerAgentName = "clusterbuildtemplate-controller"

// NewController returns a new build template controller
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {

	logger := logging.FromContext(ctx)
	kubeclientset := kubeclient.Get(ctx)
	buildclientset := buildclient.Get(ctx)
	cachingclientset := cachingclient.Get(ctx)
	clusterBuildTemplateInformer := cbtinformer.Get(ctx)
	imageInformer := imageinformer.Get(ctx)

	// Enrich the logs with controller name
	logger = logger.Named(controllerAgentName).With(zap.String(logkey.ControllerType, controllerAgentName))

	r := &Reconciler{
		kubeclientset:               kubeclientset,
		buildclientset:              buildclientset,
		cachingclientset:            cachingclientset,
		clusterBuildTemplatesLister: clusterBuildTemplateInformer.Lister(),
		imagesLister:                imageInformer.Lister(),
		Logger:                      logger,
	}
	impl := controller.NewImpl(r, logger, "ClusterBuildTemplates")

	logger.Info("Setting up event handlers")
	// Set up an event handler for when ClusterBuildTemplate resources change
	clusterBuildTemplateInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	imageInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.Filter(v1alpha1.SchemeGroupVersion.WithKind("ClusterBuildTemplate")),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})

	return impl
}
