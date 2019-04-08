/*
Copyright 2019 The Knative Authors.

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

package cloudeventslistener

import (
	"context"
	"flag"
	"reflect"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/cache"

	"github.com/knative/pkg/controller"
	"github.com/knative/pkg/logging"
	"github.com/knative/pkg/logging/logkey"
	"github.com/kubernetes/apimachinery/pkg/util/json"

	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	clientset "github.com/knative/build/pkg/client/clientset/versioned"
	buildscheme "github.com/knative/build/pkg/client/clientset/versioned/scheme"
	informers "github.com/knative/build/pkg/client/informers/externalversions/build/v1alpha1"
	listers "github.com/knative/build/pkg/client/listers/build/v1alpha1"
	"github.com/knative/build/pkg/reconciler"
	appsv1 "k8s.io/api/apps/v1"
)

const controllerAgentName = "cloudeventslistener-controller"

var (
	// The container used to accept cloud events and generate builds.
	listenerImage = flag.String("cloud-events-listener-image", "override:latest",
		"The container image for the cloud event listener.")
)

// Reconciler is the controller.Reconciler implementation for CloudEventsListener resources
type Reconciler struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// buildclientset is a clientset for our own API group
	buildclientset clientset.Interface
	// Weird slow logger
	Logger *zap.SugaredLogger
	// Listing cloud event listeners
	cloudEventsListenerLister listers.CloudEventsListenerLister
}

// Check that we implement the controller.Reconciler interface.
var _ controller.Reconciler = (*Reconciler)(nil)

func init() {
	// Add cloudeventslistener-controller types to the default Kubernetes Scheme so Events can be
	// logged for clusterbuildtemplate-controller types.
	buildscheme.AddToScheme(scheme.Scheme)
}

// NewController returns a new cloud events listener controller
func NewController(
	kubeclientset kubernetes.Interface,
	buildclientset clientset.Interface,
	logger *zap.SugaredLogger,
	cloudEventsListenerInformer informers.CloudEventsListenerInformer,
) *controller.Impl {
	// Enrich the logs with controller name
	logger = logger.Named(controllerAgentName).With(zap.String(logkey.ControllerType, controllerAgentName))

	r := &Reconciler{
		kubeclientset:             kubeclientset,
		buildclientset:            buildclientset,
		Logger:                    logger,
		cloudEventsListenerLister: cloudEventsListenerInformer.Lister(),
	}
	impl := controller.NewImpl(r, logger, "CloudEventsListener",
		reconciler.MustNewStatsReporter("CloudEventsListener", r.Logger))

	logger.Info("Setting up cloud-events-listener event handler")
	// Set up an event handler for when CloudEventsListener resources change
	cloudEventsListenerInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
	})

	return impl
}

// Reconcile will create the necessary statefulset to manage the listener process
func (c *Reconciler) Reconcile(ctx context.Context, key string) error {
	logger := logging.FromContext(ctx)
	logger.Info("cloud-events-listener reconcile")

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.Errorf("invalid resource key: %s", key)
		return nil
	}

	cel, err := c.cloudEventsListenerLister.CloudEventsListeners(namespace).Get(name)
	if errors.IsNotFound(err) {
		logger.Errorf("cloud event listener %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		return err
	}

	cel = cel.DeepCopy()
	setName := cel.Name + "-statefulset"

	buildData, err := json.Marshal(cel.Spec.Build)
	if err != nil {
		logger.Errorf("could not marshal cloudevent: %q", key)
		return err

	}

	containerArgs := []string{
		"-event-type", cel.Spec.CloudEventType,
		"-branch", cel.Spec.Branch,
		"-namespace", cel.Spec.Namespace,
	}

	logger.Infof("launching listener with args type: %s branch: %s namespace: %s", cel.Spec.CloudEventType, cel.Spec.Branch, cel.Spec.Namespace)

	secretName := "knative-cloud-event-listener-secret-" + name

	// mount a secret to house the desired build definition
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: cel.Namespace,
		},
		Data: map[string][]byte{
			"build": []byte(buildData),
		},
	}

	_, err = c.kubeclientset.Core().Secrets(cel.Namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = c.kubeclientset.Core().Secrets(cel.Namespace).Create(&secret)
			if err != nil {
				logger.Errorf("Unable to create build secret:", err)
				return err
			}

		} else {
			logger.Errorf("Unable to get build secret:", err)
			return err
		}
	}

	// Create a stateful set for the listener. It mounts a secret containing the build information.
	// The build spec may contain sensetive data and therefore the whole thing seems safest/easiest as a secret
	set := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      setName,
			Namespace: cel.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"statefulset": cel.Name + "-statefulset"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
					"statefulset": cel.Name + "-statefulset",
				}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "cloud-events-listener",
							Image: *listenerImage,
							Args:  containerArgs,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "build-volume",
									MountPath: "/root/build.json",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "build-volume",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: secretName,
								},
							},
						},
					},
				},
			},
		},
	}

	found, err := c.kubeclientset.AppsV1().StatefulSets(cel.Namespace).Get(setName, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating StatefulSet", "namespace", set.Namespace, "name", set.Name)
		created, err := c.kubeclientset.AppsV1().StatefulSets(cel.Namespace).Create(set)
		cel.Status = v1alpha1.CloudEventsListenerStatus{
			Namespace:       cel.Namespace,
			StatefulSetName: created.Name,
		}

		return err
	} else if err != nil {
		return err
	}

	if !reflect.DeepEqual(set.Spec, found.Spec) {
		found.Spec = set.Spec
		logger.Info("Updating Stateful Set", "namespace", set.Namespace, "name", set.Name)
		updated, err := c.kubeclientset.AppsV1().StatefulSets(cel.Namespace).Update(found)
		if err != nil {
			return err
		}
		cel.Status = v1alpha1.CloudEventsListenerStatus{
			Namespace:       cel.Namespace,
			StatefulSetName: updated.Name,
		}
	}
	return nil
}
