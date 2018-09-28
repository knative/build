package v1alpha1

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidateBuild(t *testing.T) {
	//ctx := context.Background()
	//hasDefault := "has-default"
	//empty := ""
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
				Timeout: metav1.Duration{Duration: -48 * time.Hour},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		reason: "maximum timeout",
		build: &Build{
			Spec: BuildSpec{
				Timeout: metav1.Duration{Duration: 48 * time.Hour},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		build: &Build{
			Spec: BuildSpec{
				Timeout: metav1.Duration{Duration: 5 * time.Minute},
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
			// client := fakekubeclientset.NewSimpleClientset()
			// buildClient := fakebuildclientset.NewSimpleClientset()
			// // Create a BuildTemplate.
			// if c.tmpl != nil {
			// 	if _, err := buildClient.BuildV1alpha1().BuildTemplates("").Create(c.tmpl); err != nil {
			// 		t.Fatalf("Failed to create BuildTemplate: %v", err)
			// 	}
			// } else if c.ctmpl != nil {
			// 	if _, err := buildClient.BuildV1alpha1().ClusterBuildTemplates().Create(c.ctmpl); err != nil {
			// 		t.Fatalf("Failed to create ClusterBuildTemplate: %v", err)
			// 	}
			// }
			// Create ServiceAccount or create the default ServiceAccount.
			// if c.sa != nil {
			// 	if _, err := client.CoreV1().ServiceAccounts(c.sa.Namespace).Create(c.sa); err != nil {
			// 		t.Fatalf("Failed to create ServiceAccount: %v", err)
			// 	}
			// } else {
			// 	if _, err := client.CoreV1().ServiceAccounts("").Create(&corev1.ServiceAccount{
			// 		ObjectMeta: metav1.ObjectMeta{Name: "default"},
			// 	}); err != nil {
			// 		t.Fatalf("Failed to create ServiceAccount: %v", err)
			// 	}
			// }
			// // Create any necessary Secrets.
			// for _, s := range c.secrets {
			// 	if _, err := client.CoreV1().Secrets("").Create(s); err != nil {
			// 		t.Fatalf("Failed to create Secret %q: %v", s.Name, err)
			// 	}
			// }

			//ac := NewAdmissionController(client, buildClient, &nop.Builder{}, defaultOptions, testLogger)
			//verr := ac.validateBuild(ctx, nil, nil, c.build)
			verr := c.build.Validate()
			if gotErr, wantErr := verr != nil, c.reason != ""; gotErr != wantErr {
				t.Errorf("validateBuild(%s); got %#v, want %q", name, verr, c.reason)
			}
		})
	}
}
