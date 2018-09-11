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
	"errors"
	"fmt"
	"testing"

	"github.com/knative/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
	buildversioned "github.com/knative/build/pkg/client/clientset/versioned"
	buildtyped "github.com/knative/build/pkg/client/clientset/versioned/typed/build/v1alpha1"
)

type clients struct {
	kubeClient  *test.KubeClient
	buildClient *buildClient
}

const buildTestNamespace = "build-tests"

func setup(t *testing.T) *clients {
	clients, err := newClients(test.Flags.Kubeconfig, test.Flags.Cluster, buildTestNamespace)
	if err != nil {
		t.Fatalf("newClients: %v", err)
	}

	return clients
}

func newClients(configPath string, clusterName string, namespace string) (*clients, error) {
	overrides := clientcmd.ConfigOverrides{}
	// Override the cluster name if provided.
	if clusterName != "" {
		overrides.Context.Cluster = clusterName
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(&clientcmd.ClientConfigLoadingRules{
		ExplicitPath: configPath,
	}, &overrides).ClientConfig()
	if err != nil {
		return nil, err
	}

	kubeClient, err := test.NewKubeClient(configPath, clusterName)
	if err != nil {
		return nil, err
	}

	bcs, err := buildversioned.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	buildClient := &buildClient{builds: bcs.BuildV1alpha1().Builds(namespace)}

	return &clients{
		kubeClient:  kubeClient,
		buildClient: buildClient,
	}, nil
}

type buildClient struct {
	builds buildtyped.BuildInterface
}

func (c *buildClient) watchBuild(name string) (*v1alpha1.Build, error) {
	ls := metav1.SingleObject(metav1.ObjectMeta{Name: name})
	// TODO: Update watchBuild function to take this as parameter depending on test requirements

	// Set build timeout to 120 seconds. This will trigger watch timeout error
	var timeout int64 = 120
	ls.TimeoutSeconds = &timeout

	w, err := c.builds.Watch(ls)
	if err != nil {
		return nil, err
	}
	for evt := range w.ResultChan() {
		switch evt.Type {
		case watch.Deleted:
			return nil, errors.New("build deleted")
		case watch.Error:
			return nil, fmt.Errorf("error event: %v", evt.Object)
		}

		b, ok := evt.Object.(*v1alpha1.Build)
		if !ok {
			return nil, fmt.Errorf("object was not a Build: %v", err)
		}

		for _, cond := range b.Status.Conditions {
			if cond.Type == v1alpha1.BuildSucceeded {
				switch cond.Status {
				case corev1.ConditionTrue:
					return b, nil
				case corev1.ConditionFalse:
					return b, errors.New("build failed")
				case corev1.ConditionUnknown:
					continue
				}
			}
		}
	}
	return nil, errors.New("watch ended before build completion")
}
