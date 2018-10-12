package build

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/knative/pkg/apis"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	logtesting "github.com/knative/pkg/logging/testing"
	"github.com/knative/pkg/signals"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
	fakebuildclientset "github.com/knative/build/pkg/client/clientset/versioned/fake"
	informers "github.com/knative/build/pkg/client/informers/externalversions"
)

const namespace = "namespace"

var ignoreVolatileTime = cmp.Comparer(func(_, _ apis.VolatileTime) bool { return true })

// TestBuildFlow creates a build and checks that a pod is created as a result.
// Then it simulates updates to the pod and checks the evolving state of the
// build in response to its pod's updates.
func TestBuildFlow(t *testing.T) {
	logger := logtesting.TestLogger(t)

	ctx := context.Background()
	buildName := "build"
	buildKey := fmt.Sprintf("%s/%s", namespace, buildName)
	kubeClient := fakeclientset.NewSimpleClientset(
		// Pre-populate ServiceAccount.
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: namespace,
			},
		})
	buildClient := fakebuildclientset.NewSimpleClientset(
		// Pre-populate a build.
		&v1alpha1.Build{
			ObjectMeta: metav1.ObjectMeta{
				Name:      buildName,
				Namespace: namespace,
			},
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "bar",
				}},
			},
		})
	stopCh := signals.SetupSignalHandler()
	informer := informers.NewSharedInformerFactory(buildClient, time.Second)
	buildsLister := informer.Build().V1alpha1().Builds()
	buildTemplatesLister := informer.Build().V1alpha1().BuildTemplates()
	clusterBuildTemplatesLister := informer.Build().V1alpha1().ClusterBuildTemplates()
	go informer.Start(stopCh)
	if ok := cache.WaitForCacheSync(stopCh, buildsLister.Informer().HasSynced); !ok {
		t.Fatalf("WaitForCacheSync failed")
	}
	rec := NewController(
		logger,
		kubeClient,
		buildClient,
		buildsLister,
		buildTemplatesLister,
		clusterBuildTemplatesLister,
	).Reconciler

	// After reconciling, the build should be updated with a pod name. That
	// pod hasn't started though.
	if err := rec.Reconcile(ctx, buildKey); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	b, err := buildClient.BuildV1alpha1().Builds(namespace).Get(buildName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Getting build %q: %v", buildName, err)
	}
	if b.Status.Cluster == nil || b.Status.Cluster.PodName == "" {
		t.Fatal("Build has no podName")
	}
	podName := b.Status.Cluster.PodName
	if diff := cmp.Diff(b.Status, v1alpha1.BuildStatus{
		Builder: v1alpha1.ClusterBuildProvider,
		Cluster: &v1alpha1.ClusterSpec{
			PodName:   podName,
			Namespace: namespace,
		},
		Conditions: []duckv1alpha1.Condition{{
			Type:   v1alpha1.BuildSucceeded,
			Status: corev1.ConditionUnknown,
		}},
	}, ignoreVolatileTime); diff != "" {
		t.Errorf("Got diff: %s", diff)
	}

	// TODO: Check state of created pod (build-step- prefixed container names, credsinit, etc.)

	// Simulate the pod starting (step hasn't started yet).
	podStart := metav1.Now()
	if _, err := kubeClient.CoreV1().Pods(namespace).Update(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Status: corev1.PodStatus{
			StartTime: &podStart,
			Phase:     corev1.PodRunning,
			InitContainerStatuses: []corev1.ContainerStatus{{
				Name: "build-step-foo",
				State: corev1.ContainerState{
					Waiting: &corev1.ContainerStateWaiting{},
				},
			}},
		},
	}); err != nil {
		t.Fatalf("Updating pod: %v", err)
	}

	// After reconciling, the build should be updated with the pod's start
	// time and step statuses.
	if err := rec.Reconcile(ctx, buildKey); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	b, err = buildClient.BuildV1alpha1().Builds(namespace).Get(buildName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Getting build %q: %v", buildName, err)
	}
	if diff := cmp.Diff(b.Status, v1alpha1.BuildStatus{
		Builder: v1alpha1.ClusterBuildProvider,
		Cluster: &v1alpha1.ClusterSpec{
			PodName:   podName,
			Namespace: namespace,
		},
		StartTime: podStart,
		Conditions: []duckv1alpha1.Condition{{
			Type:   v1alpha1.BuildSucceeded,
			Status: corev1.ConditionUnknown,
		}},
		StepStates: []corev1.ContainerState{{
			Waiting: &corev1.ContainerStateWaiting{},
		}},
	}, ignoreVolatileTime); diff != "" {
		t.Errorf("Got diff: %s", diff)
	}

	// Simulate the first step starting on the pod.
	stepStart := metav1.Now()
	if _, err := kubeClient.CoreV1().Pods(namespace).Update(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Status: corev1.PodStatus{
			StartTime: &podStart,
			Phase:     corev1.PodRunning,
			InitContainerStatuses: []corev1.ContainerStatus{{
				Name: "build-step-foo",
				State: corev1.ContainerState{
					Running: &corev1.ContainerStateRunning{
						StartedAt: stepStart,
					},
				},
			}},
		},
	}); err != nil {
		t.Fatalf("Updating pod: %v", err)
	}

	// After reconciling, the build should be updated with the pod's status
	// and step statuses.
	if err := rec.Reconcile(ctx, buildKey); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	b, err = buildClient.BuildV1alpha1().Builds(namespace).Get(buildName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Getting build %q: %v", buildName, err)
	}
	if diff := cmp.Diff(b.Status, v1alpha1.BuildStatus{
		Builder: v1alpha1.ClusterBuildProvider,
		Cluster: &v1alpha1.ClusterSpec{
			PodName:   podName,
			Namespace: namespace,
		},
		StartTime: podStart,
		Conditions: []duckv1alpha1.Condition{{
			Type:   v1alpha1.BuildSucceeded,
			Status: corev1.ConditionUnknown,
		}},
		StepStates: []corev1.ContainerState{{
			Running: &corev1.ContainerStateRunning{
				StartedAt: stepStart,
			},
		}},
	}, ignoreVolatileTime); diff != "" {
		t.Errorf("Got diff: %s", diff)
	}

	// Simulate the step and build pod finishing successfully.
	stepFinish := metav1.Now()
	if _, err := kubeClient.CoreV1().Pods(namespace).Update(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Status: corev1.PodStatus{
			StartTime: &podStart,
			Phase:     corev1.PodSucceeded,
			InitContainerStatuses: []corev1.ContainerStatus{{
				Name: "build-step-foo",
				State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{
						StartedAt:  stepStart,
						FinishedAt: stepFinish,
					},
				},
			}},
		},
	}); err != nil {
		t.Fatalf("Updating pod: %v", err)
	}

	// After reconciling, the build should be updated with the pod's status
	// and step statuses.
	if err := rec.Reconcile(ctx, buildKey); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	b, err = buildClient.BuildV1alpha1().Builds(namespace).Get(buildName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Getting build %q: %v", buildName, err)
	}
	// CompletionTime will be set by the build controller, so we won't know
	// its exact value. It should be ignored by cmp, but we'll check it's a
	// recent date.
	ignoreCompletionTime := cmp.FilterPath(func(p cmp.Path) bool { return p.String() == "CompletionTime" }, cmp.Ignore())
	if diff := cmp.Diff(b.Status, v1alpha1.BuildStatus{
		Builder: v1alpha1.ClusterBuildProvider,
		Cluster: &v1alpha1.ClusterSpec{
			PodName:   podName,
			Namespace: namespace,
		},
		StartTime: podStart,
		Conditions: []duckv1alpha1.Condition{{
			Type:   v1alpha1.BuildSucceeded,
			Status: corev1.ConditionTrue,
		}},
		StepsCompleted: []string{"build-step-foo"},
		StepStates: []corev1.ContainerState{{
			Terminated: &corev1.ContainerStateTerminated{
				StartedAt:  stepStart,
				FinishedAt: stepFinish,
			},
		}},
	}, ignoreVolatileTime, ignoreCompletionTime); diff != "" {
		t.Errorf("Got diff: %s", diff)
	}
	if b.Status.CompletionTime.Time.Before(time.Now().Add(-time.Minute)) {
		t.Errorf("Got completionTime %s, want within the last minute", b.Status.CompletionTime)
	}
}

func TestApplyTemplate(t *testing.T) {
	world := "world"
	defaultStr := "default"
	empty := ""
	for i, c := range []struct {
		build *v1alpha1.Build
		tmpl  v1alpha1.BuildTemplateInterface
		want  *v1alpha1.Build // if nil, expect error.
	}{{
		// Build's Steps are overwritten. This isn't a valid build, but
		// this code should handle it anyway.
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name: "from-build",
				}},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name: "from-template",
				}},
			},
		},
		want: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name: "from-template",
				}},
			},
		},
	}, {
		// Volumes from both build and template.
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Volumes: []corev1.Volume{{
					Name: "from-build",
				}},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Volumes: []corev1.Volume{{
					Name: "from-template",
				}},
			},
		},
		want: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Volumes: []corev1.Volume{{
					Name: "from-build",
				}, {
					Name: "from-template",
				}},
			},
		},
	}, {
		// Parameter placeholders are filled by arg value in all
		// fields.
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "FOO",
						Value: "world",
					}},
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name:  "hello ${FOO}",
					Image: "busybox:${FOO}",
					Args:  []string{"hello", "to the ${FOO}"},
					Env: []corev1.EnvVar{{
						Name:  "FOO",
						Value: "is ${FOO}",
					}},
					Command:    []string{"cmd", "${FOO}"},
					WorkingDir: "/dir/${FOO}/bar",
					VolumeMounts: []corev1.VolumeMount{{
						Name:      "${FOO}",
						MountPath: "path/to/${FOO}",
						SubPath:   "sub/${FOO}/path",
					}},
				}},
				Parameters: []v1alpha1.ParameterSpec{{
					Name: "FOO",
				}},
			},
		},
		want: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name:  "hello world",
					Image: "busybox:world",
					Args:  []string{"hello", "to the world"},
					Env: []corev1.EnvVar{{
						Name:  "FOO",
						Value: "is world",
					}},
					Command:    []string{"cmd", "world"},
					WorkingDir: "/dir/world/bar",
					VolumeMounts: []corev1.VolumeMount{{
						Name:      "world",
						MountPath: "path/to/world",
						SubPath:   "sub/world/path",
					}},
				}},
				Template: &v1alpha1.TemplateInstantiationSpec{
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "FOO",
						Value: "world",
					}},
				},
			},
		},
	}, {
		// $-prefixed strings (e.g., env vars in a bash script) are untouched.
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "FOO",
						Value: "world",
					}},
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name:    "ubuntu",
					Command: []string{"bash"},
					Args:    []string{"-c", "echo $BAR ${FOO}"},
					Env: []corev1.EnvVar{{
						Name:  "BAR",
						Value: "terrible",
					}},
				}},
				Parameters: []v1alpha1.ParameterSpec{{
					Name: "FOO",
				}},
			},
		},
		want: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name:    "ubuntu",
					Command: []string{"bash"},
					Args:    []string{"-c", "echo $BAR world"},
					Env: []corev1.EnvVar{{
						Name:  "BAR",
						Value: "terrible",
					}},
				}},
				Template: &v1alpha1.TemplateInstantiationSpec{
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "FOO",
						Value: "world",
					}},
				},
			},
		},
	}, {
		// $$-prefixed strings are untouched, even if they conflict with a
		// parameter name.
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "FOO",
						Value: "world",
					}},
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name:    "ubuntu",
					Command: []string{"bash"},
					Args:    []string{"-c", "echo $FOO ${FOO}"},
					Env: []corev1.EnvVar{{
						Name:  "FOO",
						Value: "terrible",
					}},
				}},
				Parameters: []v1alpha1.ParameterSpec{{
					Name: "FOO",
				}},
			},
		},
		want: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name:    "ubuntu",
					Command: []string{"bash"},
					Args:    []string{"-c", "echo $FOO world"},
					Env: []corev1.EnvVar{{
						Name:  "FOO",
						Value: "terrible",
					}},
				}},
				Template: &v1alpha1.TemplateInstantiationSpec{
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "FOO",
						Value: "world",
					}},
				},
			},
		},
	}, {
		// Parameter with default value.
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name:  "hello ${FOO}",
					Image: "busybox:${FOO}",
					Args:  []string{"hello", "to the ${FOO}"},
					Env: []corev1.EnvVar{{
						Name:  "FOO",
						Value: "is ${FOO}",
					}},
					Command:    []string{"cmd", "${FOO}"},
					WorkingDir: "/dir/${FOO}/bar",
				}},
				Parameters: []v1alpha1.ParameterSpec{{
					Name:    "FOO",
					Default: &world,
				}},
			},
		},
		want: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name:  "hello world",
					Image: "busybox:world",
					Args:  []string{"hello", "to the world"},
					Env: []corev1.EnvVar{{
						Name:  "FOO",
						Value: "is world",
					}},
					Command:    []string{"cmd", "world"},
					WorkingDir: "/dir/world/bar",
				}},
			},
		},
	}, {
		// Parameter with empty default value.
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name: "hello ${FOO}",
					Args: []string{"hello", "to the ${FOO}"},
					Env: []corev1.EnvVar{{
						Name:  "FOO",
						Value: "is ${FOO}",
					}},
					Command:    []string{"cmd", "${FOO}"},
					WorkingDir: "/dir/${FOO}/bar",
				}},
				Parameters: []v1alpha1.ParameterSpec{{
					Name:    "FOO",
					Default: &empty,
				}},
			},
		},
		want: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name: "hello ",
					Args: []string{"hello", "to the "},
					Env: []corev1.EnvVar{{
						Name:  "FOO",
						Value: "is ",
					}},
					Command:    []string{"cmd", ""},
					WorkingDir: "/dir//bar",
				}},
			},
		},
	}, {
		// Parameter with default value, which build overrides.
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "FOO",
						Value: "world",
					}},
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name: "hello ${FOO}",
					Args: []string{"hello", "to the ${FOO}"},
					Env: []corev1.EnvVar{{
						Name:  "FOO",
						Value: "is ${FOO}",
					}},
					Command:    []string{"cmd", "${FOO}"},
					WorkingDir: "/dir/${FOO}/bar",
				}},
				Parameters: []v1alpha1.ParameterSpec{{
					Name:    "FOO",
					Default: &defaultStr,
				}},
			},
		},
		want: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name: "hello world",
					Args: []string{"hello", "to the world"},
					Env: []corev1.EnvVar{{
						Name:  "FOO",
						Value: "is world",
					}},
					Command:    []string{"cmd", "world"},
					WorkingDir: "/dir/world/bar",
				}},
				Template: &v1alpha1.TemplateInstantiationSpec{
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "FOO",
						Value: "world",
					}},
				},
			},
		},
	}, {
		// Unsatisfied parameter (no default), so it's not replaced.
		// This doesn't pass ValidateBuild anyway.
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name: "hello ${FOO}",
					Args: []string{"hello", "to the ${FOO}"},
					Env: []corev1.EnvVar{{
						Name:  "FOO",
						Value: "is ${FOO}",
					}},
					Command:    []string{"cmd", "${FOO}"},
					WorkingDir: "/dir/${FOO}/bar",
				}},
				Parameters: []v1alpha1.ParameterSpec{{
					Name: "FOO",
				}},
			},
		},
		want: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name: "hello ${FOO}",
					Args: []string{"hello", "to the ${FOO}"},
					Env: []corev1.EnvVar{{
						Name:  "FOO",
						Value: "is ${FOO}",
					}},
					Command:    []string{"cmd", "${FOO}"},
					WorkingDir: "/dir/${FOO}/bar",
				}},
			},
		},
	}, {
		// Build with arg for unknown param (ignored).
		// TODO(jasonhall): Should this be an error?
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "FOO",
						Value: "world",
					}},
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name: "hello",
				}},
			},
		},
		want: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name: "hello",
				}},
				Template: &v1alpha1.TemplateInstantiationSpec{
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "FOO",
						Value: "world",
					}},
				},
			},
		},
	}, {
		// Template doesn't specify that ${FOO} is a parameter, so it's not
		// replaced.
		// TODO(jasonhall): Should this be an error?
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name: "hello ${FOO}",
					Args: []string{"hello", "to the ${FOO}"},
					Env: []corev1.EnvVar{{
						Name:  "FOO",
						Value: "is ${FOO}",
					}},
					Command:    []string{"cmd", "${FOO}"},
					WorkingDir: "/dir/${FOO}/bar",
				}},
			},
		},
		want: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name: "hello ${FOO}",
					Args: []string{"hello", "to the ${FOO}"},
					Env: []corev1.EnvVar{{
						Name:  "FOO",
						Value: "is ${FOO}",
					}},
					Command:    []string{"cmd", "${FOO}"},
					WorkingDir: "/dir/${FOO}/bar",
				}},
			},
		},
	}, {
		// Malformed placeholders are ignored.
		// TODO(jasonhall): Should this be an error?
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "FOO",
						Value: "world",
					}},
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name: "hello ${FOO",
				}},
				Parameters: []v1alpha1.ParameterSpec{{
					Name: "FOO",
				}},
			},
		},
		want: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name: "hello ${FOO",
				}},
				Template: &v1alpha1.TemplateInstantiationSpec{
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "FOO",
						Value: "world",
					}},
				},
			},
		},
	}, {
		// A build's template initiation spec contains
		// env vars
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Env: []corev1.EnvVar{{
						Name:  "SOME_ENV_VAR",
						Value: "foo",
					}},
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name: "hello",
				}},
			},
		},
		want: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name: "hello",
					Env: []corev1.EnvVar{{
						Name:  "SOME_ENV_VAR",
						Value: "foo",
					}},
				}},
				Template: &v1alpha1.TemplateInstantiationSpec{
					Env: []corev1.EnvVar{{
						Name:  "SOME_ENV_VAR",
						Value: "foo",
					}},
				},
			},
		},
	}, {
		// A cluster build template
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Kind: v1alpha1.ClusterBuildTemplateKind,
				},
			},
		},
		tmpl: &v1alpha1.ClusterBuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name: "hello",
				}},
			},
		},
		want: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name: "hello",
				}},
				Template: &v1alpha1.TemplateInstantiationSpec{
					Kind: v1alpha1.ClusterBuildTemplateKind,
				},
			},
		},
	}, {
		// A build template with kind BuildTemplate
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Kind: v1alpha1.BuildTemplateKind,
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name: "hello",
				}},
			},
		},
		want: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name: "hello",
				}},
				Template: &v1alpha1.TemplateInstantiationSpec{
					Kind: v1alpha1.BuildTemplateKind,
				},
			},
		},
	}} {
		wantErr := c.want == nil
		got, err := applyTemplate(c.build, c.tmpl)
		if err != nil && !wantErr {
			t.Errorf("applyTemplate(%d); unexpected error %v", i, err)
		} else if err == nil && wantErr {
			t.Errorf("applyTemplate(%d); unexpected success; got %v", i, got)
		} else if !reflect.DeepEqual(got, c.want) {
			t.Errorf("applyTemplate(%d);\n got %v\nwant %v", i, got, c.want)
		}
	}
}
