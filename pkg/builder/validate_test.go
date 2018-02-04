/*
Copyright 2018 Google, Inc. All rights reserved.

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

package builder

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	v1alpha1 "github.com/google/build-crd/pkg/apis/cloudbuild/v1alpha1"
)

func TestValidateBuild(t *testing.T) {
	tests := []struct {
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
		// Multiple unnamed steps.
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
					Name: "my-template",
				},
			},
		},
		reason: "TemplateAndSteps",
	}, {
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Name: "foo-bar",
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{},
			Status: v1alpha1.BuildTemplateStatus{
				Conditions: []v1alpha1.BuildTemplateCondition{{
					Type:   v1alpha1.BuildTemplateInvalid,
					Status: corev1.ConditionTrue,
				}},
			},
		},
		reason: "InvalidTemplate",
	}, {
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "foo",
						Value: "hello",
					}},
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
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
		// valid, arg doesn't match any parameter.
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Name: "foo-bar",
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "bar",
						Value: "hello",
					}},
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{},
		},
	}, {
		// valid, since unsatisfied parameter has a default.
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{
					Name: "foo-bar",
					Arguments: []v1alpha1.ArgumentSpec{{
						Name:  "foo",
						Value: "hello",
					}},
				},
			},
		},
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Parameters: []v1alpha1.ParameterSpec{{
					Name: "foo",
				}, {
					Name:    "bar",
					Default: "has-default",
				}},
			},
		},
	}, {
		// invalid, since build is missing template name
		build: &v1alpha1.Build{
			Spec: v1alpha1.BuildSpec{
				Template: &v1alpha1.TemplateInstantiationSpec{},
			},
		},
		reason: "MissingTemplateName",
	}}
	for i, test := range tests {
		verr := ValidateBuild(test.build, test.tmpl)
		if gotErr, wantErr := verr != nil, test.reason != ""; gotErr != wantErr {
			t.Errorf("ValidateBuild(%d); got %v, want %q", i, verr, test.reason)
		}
	}
}

func TestValidateTemplate(t *testing.T) {
	tests := []struct {
		tmpl   *v1alpha1.BuildTemplate
		reason string // if "", expect success.
	}{{
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		// Multiple unnamed steps.
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
		// invalid, template step name has nested placeholder.
		tmpl: &v1alpha1.BuildTemplate{
			Spec: v1alpha1.BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name: "step-name-${FOO${BAR}}",
				}},
				Parameters: []v1alpha1.ParameterSpec{{
					Name: "FOO",
				}, {
					Name:    "BAR",
					Default: "has-default",
				}},
			},
		},
		reason: "NestedPlaceholder",
	}}
	for i, test := range tests {
		verr := ValidateTemplate(test.tmpl)
		if gotErr, wantErr := verr != nil, test.reason != ""; gotErr != wantErr {
			t.Errorf("ValidateTemplate(%d); got %v, want %q", i, verr, test.reason)
		}
	}
}
