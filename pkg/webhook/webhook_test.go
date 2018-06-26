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

package webhook

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mattbaird/jsonpatch"
	"go.uber.org/zap"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/knative/build/pkg/builder/nop"
	fakebuildclientset "github.com/knative/build/pkg/client/clientset/versioned/fake"
	"github.com/knative/build/pkg/logging"
)

const (
	testNamespace         = "test-namespace"
	testBuildName         = "test-build"
	testBuildTemplateName = "test-build-template"
)

var (
	defaultOptions = ControllerOptions{
		ServiceName:      "build-webhook",
		ServiceNamespace: "knative-build",
		Port:             443,
		SecretName:       "build-webhook-certs",
		WebhookName:      "webhook.build.knative.dev",
	}
	testLogger = zap.NewNop().Sugar()
	testCtx    = logging.WithLogger(context.TODO(), testLogger)
)

func testBuild(name string) v1alpha1.Build {
	return v1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      name,
		},
		Spec: v1alpha1.BuildSpec{},
	}
}

func testBuildTemplate(name string) v1alpha1.BuildTemplate {
	return v1alpha1.BuildTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      name,
		},
		Spec: v1alpha1.BuildTemplateSpec{},
	}
}

func mustMarshal(t *testing.T, in interface{}) []byte {
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	return b
}

func mustUnmarshalPatches(t *testing.T, b []byte) []jsonpatch.JsonPatchOperation {
	var p []jsonpatch.JsonPatchOperation
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatalf("Unmarshalling patches: %v", err)
	}
	return p
}

func TestAdmitBuild(t *testing.T) {
	for _, c := range []struct {
		desc        string
		op          admissionv1beta1.Operation
		kind        string
		wantAllowed bool
		new, old    v1alpha1.Build
		wantPatches []jsonpatch.JsonPatchOperation
	}{{
		desc:        "delete op",
		op:          admissionv1beta1.Delete,
		wantAllowed: true,
	}, {
		desc:        "connect op",
		op:          admissionv1beta1.Connect,
		wantAllowed: true,
	}, {
		desc:        "bad kind",
		op:          admissionv1beta1.Create,
		kind:        "Garbage",
		wantAllowed: false,
	}, {
		desc:        "invalid name",
		op:          admissionv1beta1.Create,
		kind:        "Build",
		new:         testBuild("build.invalid"),
		wantAllowed: false,
	}, {
		desc:        "invalid name too long",
		op:          admissionv1beta1.Create,
		kind:        "Build",
		new:         testBuild(strings.Repeat("a", 64)),
		wantAllowed: false,
	}, {
		desc:        "create valid",
		op:          admissionv1beta1.Create,
		kind:        "Build",
		new:         testBuild("valid-build"),
		wantAllowed: true,
		wantPatches: []jsonpatch.JsonPatchOperation{{
			Operation: "add",
			Path:      "/spec/generation",
			Value:     float64(1),
		}},
	}, {
		desc:        "no-op update",
		op:          admissionv1beta1.Update,
		kind:        "Build",
		old:         testBuild("valid-build"),
		new:         testBuild("valid-build"),
		wantAllowed: true,
		wantPatches: nil,
	}} {
		t.Run(c.desc, func(t *testing.T) {
			ctx := context.Background()
			ac := NewAdmissionController(fakekubeclientset.NewSimpleClientset(), fakebuildclientset.NewSimpleClientset(), &nop.Builder{}, defaultOptions, testLogger)
			resp := ac.admit(ctx, &admissionv1beta1.AdmissionRequest{
				Operation: c.op,
				Kind:      metav1.GroupVersionKind{Kind: c.kind},
				OldObject: runtime.RawExtension{Raw: mustMarshal(t, c.old)},
				Object:    runtime.RawExtension{Raw: mustMarshal(t, c.new)},
			})
			if resp.Allowed != c.wantAllowed {
				t.Errorf("allowed got %t, want %t", resp.Allowed, c.wantAllowed)
			}
			if c.wantPatches != nil {
				gotPatches := mustUnmarshalPatches(t, resp.Patch)
				if diff := cmp.Diff(gotPatches, c.wantPatches); diff != "" {
					t.Errorf("patches differed: %s", diff)
				}
			}
		})
	}
}

func TestValidateBuild(t *testing.T) {
	ctx := context.Background()
	hasDefault := "has-default"
	empty := ""
	for _, c := range []struct {
		desc   string
		build  *v1alpha1.Build
		tmpl   *v1alpha1.BuildTemplate
		reason string // if "", expect success.
	}{{
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		desc: "Multiple unnamed steps",
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Image: "gcr.io/foo-bar/baz:latest",
				}, {
					Image: "gcr.io/foo-bar/baz:latest",
				}, {
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}, {
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:oops",
				}},
			},
		},
		reason: "DuplicateStepName",
	}, {
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
				Volumes: []corev1.Volume{{
					Name: "foo",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				}, {
					Name: "foo",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				}},
			},
		},
		reason: "DuplicateVolumeName",
	}, {
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "foo",
						Value: "hello",
					}, {
						Name:  "foo",
						Value: "world",
					}},
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Parameters: []v1alpha1.ParameterSpec{{
					Name: "foo",
				}},
			},
		},
		reason: "DuplicateArgName",
	}, {
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Name: "foo-bar",
				},
				Volumes: []corev1.Volume{{
					Name: "foo",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				}},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
				Volumes: []corev1.Volume{{
					Name: "foo",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				}},
			},
		},
		reason: "DuplicateVolumeName",
	}, {
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
				Template: &v1alpha1.TemplateInstantiationSpec{
					Name: "template",
				},
			},
		},
		reason: "TemplateAndSteps",
	}, {
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Name: "template",
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "foo",
						Value: "hello",
					}},
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "template"},
			Spec: v1alpha1.BuildTemplateSpec{
				Parameters: []v1alpha1.ParameterSpec{{
					Name: "foo",
				}, {
					Name: "bar",
				}},
			},
		},
		reason: "UnsatisfiedParameter",
	}, {
		desc: "Arg doesn't match any parameter",
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Name: "template",
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "bar",
						Value: "hello",
					}},
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "template"},
			Spec:       v1alpha1.BuildTemplateSpec{},
		},
	}, {
		desc: "Unsatisfied parameter has a default",
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Name: "template",
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "foo",
						Value: "hello",
					}},
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "template"},
			Spec: v1alpha1.BuildTemplateSpec{
				Parameters: []v1alpha1.ParameterSpec{{
					Name: "foo",
				}, {
					Name:    "bar",
					Default: &hasDefault,
				}},
			},
		},
	}, {
		desc: "Unsatisfied parameter has empty default",
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Name: "empty-default",
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			ObjectMeta: metav1.ObjectMeta{Name: "empty-default"},
			Spec: v1alpha1.BuildTemplateSpec{
				Parameters: []v1alpha1.ParameterSpec{{
					Name:    "foo",
					Default: &empty,
				}},
			},
		},
	}, {
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{},
			},
		},
		reason: "MissingTemplateName",
	}} {
		name := c.desc
		if c.reason != "" {
			name = "invalid-" + c.reason
		}
		t.Run(name, func(t *testing.T) {
			buildClient := fakebuildclientset.NewSimpleClientset()
			if c.tmpl != nil {
				if _, err := buildClient.BuildV1alpha1().BuildTemplates("").Create(c.tmpl); err != nil {
					t.Fatalf("Failed to create template: %v", err)
				}
			}

			ac := NewAdmissionController(fakekubeclientset.NewSimpleClientset(), buildClient, &nop.Builder{}, defaultOptions, testLogger)
			verr := ac.validateBuild(ctx, nil, nil, c.build)
			if gotErr, wantErr := verr != nil, c.reason != ""; gotErr != wantErr {
				t.Errorf("validateBuild(%s); got %v, want %q", name, verr, c.reason)
			}
		})
	}
}

func TestValidateTemplate(t *testing.T) {
	ctx := context.Background()
	hasDefault := "has-default"
	for _, c := range []struct {
		desc   string
		tmpl   *v1alpha1.BuildTemplate
		reason string // if "", expect success.
	}{{
		desc: "Single named step",
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		desc: "Multiple unnamed steps",
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Image: "gcr.io/foo-bar/baz:latest",
				}, {
					Image: "gcr.io/foo-bar/baz:latest",
				}, {
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}, {
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:oops",
				}},
			},
		},
		reason: "DuplicateStepName",
	}, {
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
				Volumes: []corev1.Volume{{
					Name: "foo",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				}, {
					Name: "foo",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				}},
			},
		},
		reason: "DuplicateVolumeName",
	}, {
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Parameters: []v1alpha1.ParameterSpec{{
					Name: "foo",
				}, {
					Name: "foo",
				}},
			},
		},
		reason: "DuplicateParamName",
	}, {
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name: "step-name-${FOO${BAR}}",
				}},
				Parameters: []v1alpha1.ParameterSpec{{
					Name: "FOO",
				}, {
					Name:    "BAR",
					Default: &hasDefault,
				}},
			},
		},
		reason: "NestedPlaceholder",
	}} {
		name := c.desc
		if c.reason != "" {
			name = "invalid-" + c.reason
		}
		t.Run(name, func(t *testing.T) {
			ac := NewAdmissionController(fakekubeclientset.NewSimpleClientset(), fakebuildclientset.NewSimpleClientset(), &nop.Builder{}, defaultOptions, testLogger)
			verr := ac.validateBuildTemplate(ctx, nil, nil, c.tmpl)
			if gotErr, wantErr := verr != nil, c.reason != ""; gotErr != wantErr {
				t.Errorf("validateBuildTemplate(%s); got %v, want %q", name, verr, c.reason)
			}
		})
	}
}

func TestAdmitBuildTemplate(t *testing.T) {
	for _, c := range []struct {
		desc        string
		op          admissionv1beta1.Operation
		kind        string
		wantAllowed bool
		new, old    v1alpha1.BuildTemplate
		wantPatches []jsonpatch.JsonPatchOperation
	}{{
		desc:        "delete op",
		op:          admissionv1beta1.Delete,
		wantAllowed: true,
	}, {
		desc:        "connect op",
		op:          admissionv1beta1.Connect,
		wantAllowed: true,
	}, {
		desc:        "bad kind",
		op:          admissionv1beta1.Create,
		kind:        "Garbage",
		wantAllowed: false,
	}, {
		desc:        "invalid name",
		op:          admissionv1beta1.Create,
		kind:        "BuildTemplate",
		new:         testBuildTemplate("build-template.invalid"),
		wantAllowed: false,
	}, {
		desc:        "invalid name too long",
		op:          admissionv1beta1.Create,
		kind:        "BuildTemplate",
		new:         testBuildTemplate(strings.Repeat("a", 64)),
		wantAllowed: false,
	}, {
		desc:        "create valid",
		op:          admissionv1beta1.Create,
		kind:        "BuildTemplate",
		new:         testBuildTemplate("valid-build-template"),
		wantAllowed: true,
		wantPatches: []jsonpatch.JsonPatchOperation{{
			Operation: "add",
			Path:      "/spec/generation",
			Value:     float64(1),
		}},
	}, {
		desc:        "no-op update",
		op:          admissionv1beta1.Update,
		kind:        "BuildTemplate",
		old:         testBuildTemplate("valid-build-template"),
		new:         testBuildTemplate("valid-build-template"),
		wantAllowed: true,
		wantPatches: nil,
	}} {
		t.Run(c.desc, func(t *testing.T) {
			ctx := context.Background()
			ac := NewAdmissionController(fakekubeclientset.NewSimpleClientset(), fakebuildclientset.NewSimpleClientset(), &nop.Builder{}, defaultOptions, testLogger)
			resp := ac.admit(ctx, &admissionv1beta1.AdmissionRequest{
				Operation: c.op,
				Kind:      metav1.GroupVersionKind{Kind: c.kind},
				OldObject: runtime.RawExtension{Raw: mustMarshal(t, c.old)},
				Object:    runtime.RawExtension{Raw: mustMarshal(t, c.new)},
			})
			if resp.Allowed != c.wantAllowed {
				t.Errorf("allowed got %t, want %t", resp.Allowed, c.wantAllowed)
			}
			if c.wantPatches != nil {
				gotPatches := mustUnmarshalPatches(t, resp.Patch)
				if diff := cmp.Diff(gotPatches, c.wantPatches); diff != "" {
					t.Errorf("patches differed: %s", diff)
				}
			}
		})
	}
}
