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

package v1alpha1

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateBuild(t *testing.T) {
	for _, c := range []struct {
		desc   string
		build  *Build
		reason string // if "", expect success.
	}{{
		build: &Build{
			Spec: BuildSpec{
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		reason: "negative build timeout",
		build: &Build{
			Spec: BuildSpec{
				Timeout: &metav1.Duration{Duration: -48 * time.Hour},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		desc:   "No template and steps",
		reason: "no template & steps",
		build: &Build{
			Spec: BuildSpec{},
		},
	}, {
		desc:   "Bad template kind",
		reason: "invalid template",
		build: &Build{
			Spec: BuildSpec{
				Template: &TemplateInstantiationSpec{
					Kind: "bad-kind",
					Name: "bad-tmpl",
				},
			},
		},
	}, {
		desc: "good template kind",
		build: &Build{
			Spec: BuildSpec{
				Template: &TemplateInstantiationSpec{
					Kind: ClusterBuildTemplateKind,
					Name: "goo-tmpl",
				},
			},
		},
	}, {
		reason: "maximum timeout",
		build: &Build{
			Spec: BuildSpec{
				Timeout: &metav1.Duration{Duration: 48 * time.Hour},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		build: &Build{
			Spec: BuildSpec{
				Timeout: &metav1.Duration{Duration: 5 * time.Minute},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		desc: "Multiple unnamed steps",
		build: &Build{
			Spec: BuildSpec{
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
		build: &Build{
			Spec: BuildSpec{
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
		build: &Build{
			Spec: BuildSpec{
				Steps: []corev1.Container{{Name: "foo"}},
			},
		},
		reason: "StepMissingImage",
	}, {
		build: &Build{
			Spec: BuildSpec{
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
		build: &Build{
			Spec: BuildSpec{
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
				Template: &TemplateInstantiationSpec{
					Name: "template",
				},
			},
		},
		reason: "TemplateAndSteps",
	}, {
		build: &Build{
			Spec: BuildSpec{
				Template: &TemplateInstantiationSpec{},
			},
		},
		reason: "MissingTemplateName",
	}} {
		name := c.desc
		if c.reason != "" {
			name = "invalid-" + c.reason
		}
		t.Run(name, func(t *testing.T) {
			verr := c.build.Validate()
			if gotErr, wantErr := verr != nil, c.reason != ""; gotErr != wantErr {
				t.Errorf("validateBuild(%s); got %#v, want %q", name, verr, c.reason)
			}
		})
	}
}
