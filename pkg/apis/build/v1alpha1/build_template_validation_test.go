package v1alpha1

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestValidateTemplate(t *testing.T) {
	hasDefault := "has-default"
	for _, c := range []struct {
		desc   string
		tmpl   *BuildTemplate
		reason string // if "", expect success.
	}{{
		desc: "Single named step",
		tmpl: &BuildTemplate{
			Spec: BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		desc: "Multiple unnamed steps",
		tmpl: &BuildTemplate{
			Spec: BuildTemplateSpec{
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
		tmpl: &BuildTemplate{
			Spec: BuildTemplateSpec{
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
		tmpl: &BuildTemplate{
			Spec: BuildTemplateSpec{
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
		tmpl: &BuildTemplate{
			Spec: BuildTemplateSpec{
				Parameters: []ParameterSpec{{
					Name: "foo",
				}, {
					Name: "foo",
				}},
			},
		},
		reason: "DuplicateParamName",
	}, {
		tmpl: &BuildTemplate{
			Spec: BuildTemplateSpec{
				Steps: []corev1.Container{{
					Name: "step-name-${FOO${BAR}}",
				}},
				Parameters: []ParameterSpec{{
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
			verr := c.tmpl.Validate()
			if gotErr, wantErr := verr != nil, c.reason != ""; gotErr != wantErr {
				t.Errorf("validateBuildTemplate(%s); got %v, want %q", name, verr, c.reason)
			}
		})
	}
}
