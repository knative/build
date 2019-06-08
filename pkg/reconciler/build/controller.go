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

	buildclient "github.com/knative/build/pkg/client/injection/client"
	buildinformer "github.com/knative/build/pkg/client/injection/informers/build/v1alpha1/build"
	buildtemplateinformer "github.com/knative/build/pkg/client/injection/informers/build/v1alpha1/buildtemplate"
	clusterbuildtemplateinformer "github.com/knative/build/pkg/client/injection/informers/build/v1alpha1/clusterbuildtemplate"
	"github.com/knative/pkg/injection/clients/kubeclient"
	podinformer "github.com/knative/pkg/injection/informers/kubeinformers/corev1/pod"

	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/knative/pkg/configmap"
	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/logging/logkey"
	"go.uber.org/zap"
	"k8s.io/client-go/tools/cache"
)

const (
	controllerAgentName = "build-controller"
)

// NewController returns a new build controller
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {

	logger := logging.FromContext(ctx)
	kubeclientset := kubeclient.Get(ctx)
	buildclientset := buildclient.Get(ctx)
	podInformer := podinformer.Get(ctx)
	buildInformer := buildinformer.Get(ctx)
	buildTemplateInformer := buildtemplateinformer.Get(ctx)
	clusterBuildTemplateInformer := clusterbuildtemplateinformer.Get(ctx)

	timeoutHandler := NewTimeoutHandler(logger, kubeclientset, buildclientset, ctx.Done())
	timeoutHandler.CheckTimeouts()

	// Enrich the logs with controller name
	logger = logger.Named(controllerAgentName).With(zap.String(logkey.ControllerType, controllerAgentName))

	r := &Reconciler{
		kubeclientset:               kubeclientset,
		buildclientset:              buildclientset,
		buildsLister:                buildInformer.Lister(),
		buildTemplatesLister:        buildTemplateInformer.Lister(),
		clusterBuildTemplatesLister: clusterBuildTemplateInformer.Lister(),
		podsLister:                  podInformer.Lister(),
		Logger:                      logger,
		timeoutHandler:              timeoutHandler,
	}
	impl := controller.NewImpl(r, logger, "Builds")

	logger.Info("Setting up event handlers")
	// Set up an event handler for when Build resources change
	buildInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	// Set up a Pod informer, so that Pod updates trigger Build
	// reconciliations.
	podInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.Filter(v1alpha1.SchemeGroupVersion.WithKind("Build")),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})

	return impl
}
