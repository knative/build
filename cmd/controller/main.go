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

package main

import (
	"flag"
	"time"

	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	onclusterbuilder "github.com/knative/build/pkg/builder/cluster"
	"github.com/knative/build/pkg/controller"
	"github.com/knative/build/pkg/controller/build"
	"github.com/knative/build/pkg/controller/buildtemplate"
	"github.com/knative/build/pkg/controller/clusterbuildtemplate"
	"github.com/knative/build/pkg/logging"

	buildclientset "github.com/knative/build/pkg/client/clientset/versioned"
	informers "github.com/knative/build/pkg/client/informers/externalversions"
	"github.com/knative/pkg/signals"
)

const threadsPerController = 2

var (
	kubeconfig = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	masterURL  = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
)

func main() {
	flag.Parse()
	logger := logging.NewLoggerFromDefaultConfigMap("loglevel.controller").Named("controller")
	defer logger.Sync()

	logger.Info("Starting the Build Controller")

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeconfig)
	if err != nil {
		logger.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	buildClient, err := buildclientset.NewForConfig(cfg)
	if err != nil {
		logger.Fatalf("Error building Build clientset: %s", err.Error())
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	buildInformerFactory := informers.NewSharedInformerFactory(buildClient, time.Second*30)

	// Add new controllers here.
	ctors := []controller.Constructor{
		build.NewController,
		buildtemplate.NewController,
		clusterbuildtemplate.NewController,
	}

	bldr := onclusterbuilder.NewBuilder(kubeClient, kubeInformerFactory, logger)

	// Build all of our controllers, with the clients constructed above.
	controllers := make([]controller.Interface, 0, len(ctors))
	for _, ctor := range ctors {
		controllers = append(controllers,
			ctor(bldr, kubeClient, buildClient, kubeInformerFactory, buildInformerFactory, logger))
	}

	go kubeInformerFactory.Start(stopCh)
	go buildInformerFactory.Start(stopCh)

	// Start all of the controllers.
	for _, ctrlr := range controllers {
		go func(ctrlr controller.Interface) {
			// We don't expect this to return until stop is called,
			// but if it does, propagate it back.
			if err := ctrlr.Run(threadsPerController, stopCh); err != nil {
				logger.Fatalf("Error running controller: %s", err.Error())
			}
		}(ctrlr)
	}

	// TODO(mattmoor): Use a sync.WaitGroup instead?
	<-stopCh
}
