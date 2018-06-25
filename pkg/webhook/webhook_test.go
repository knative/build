/*
Copyright 2018 Knative Authors, Inc. All rights reserved.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
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
		ServiceNamespace: "build-system",
		Port:             443,
		SecretName:       "build-webhook-certs",
		WebhookName:      "webhook.build.dev",
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
			ac := NewAdmissionController(fakekubeclientset.NewSimpleClientset(), defaultOptions, testLogger)
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
			ac := NewAdmissionController(fakekubeclientset.NewSimpleClientset(), defaultOptions, testLogger)
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
