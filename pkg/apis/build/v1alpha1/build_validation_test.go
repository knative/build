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
		desc: "source and sources presence",
		build: &Build{
			Spec: BuildSpec{
				Sources: []SourceSpec{{
					Name: "sources",
					Git: &GitSourceSpec{
						Url:      "someurl",
						Revision: "revision",
					},
				}},
				Source: &SourceSpec{
					Git: &GitSourceSpec{
						Url:      "someurl",
						Revision: "revision",
					},
				},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
		reason: "source and sources cannot be declared in same build",
	}, {
		desc: "source without name",
		build: &Build{
			Spec: BuildSpec{
				Sources: []SourceSpec{{
					Git: &GitSourceSpec{
						Url:      "someurl",
						Revision: "revision",
					},
				}},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		desc: "source with targetPath",
		build: &Build{
			Spec: BuildSpec{
				Sources: []SourceSpec{{
					TargetPath: "/path/a/b",
					Git: &GitSourceSpec{
						Url:      "someurl",
						Revision: "revision",
					},
				}},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		desc: "sources with empty targetPaths",
		build: &Build{
			Spec: BuildSpec{
				Sources: []SourceSpec{{
					Name:       "gitpathab",
					TargetPath: "/path/a/b",
					Git: &GitSourceSpec{
						Url:      "someurl",
						Revision: "revision",
					},
				}, {
					Name: "gcsnopath1",
					GCS: &GCSSourceSpec{
						Type:     GCSArchive,
						Location: "blah",
					},
				}, {
					Name: "gcsnopath", // 2 sources with empty target path
					GCS: &GCSSourceSpec{
						Type:     GCSArchive,
						Location: "blah",
					},
				}},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
		reason: "multiple sources with empty target paths",
	}, {
		desc: "custom sources with targetPaths",
		build: &Build{
			Spec: BuildSpec{
				Sources: []SourceSpec{{
					Name:       "customwithpath",
					TargetPath: "a/b",
					Custom: &corev1.Container{
						Image: "soemthing:latest",
					},
				}},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
		reason: "custom sources with targetPaths",
	}, {
		desc: "multiple custom sources without targetPaths",
		build: &Build{
			Spec: BuildSpec{
				Sources: []SourceSpec{{
					Name: "customwithpath",
					Custom: &corev1.Container{
						Image: "soemthing:latest",
					},
				}, {
					Name: "customwithpath1",
					Custom: &corev1.Container{
						Image: "soemthing:latest",
					},
				}},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		desc: "sources with combination of different targetPath with common parent dir",
		build: &Build{
			Spec: BuildSpec{
				Sources: []SourceSpec{{
					Name:       "gitpathab",
					TargetPath: "/a/b",
					Git: &GitSourceSpec{
						Url:      "someurl",
						Revision: "revision",
					},
				}, {
					Name:       "gcsnonestedpath",
					TargetPath: "/a/b/c",
					GCS: &GCSSourceSpec{
						Type:     GCSArchive,
						Location: "blah",
					},
				}},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
		reason: "multiple sources with overlap of target paths",
	}, {
		desc: "sources with combination of individual targetpath",
		build: &Build{
			Spec: BuildSpec{
				Sources: []SourceSpec{{
					Name:       "gitpathab",
					TargetPath: "basel",
					Git: &GitSourceSpec{
						Url:      "someurl",
						Revision: "revision",
					},
				}, {
					Name:       "gcsnonestedpath",
					TargetPath: "baselrocks",
					GCS: &GCSSourceSpec{
						Type:     GCSArchive,
						Location: "blah",
					},
				}},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		desc: "Mix of sources with and without target path",
		build: &Build{
			Spec: BuildSpec{
				Sources: []SourceSpec{{
					Name:       "gitpathab",
					TargetPath: "gitpath",
					Git: &GitSourceSpec{
						Url:      "someurl",
						Revision: "revision",
					},
				}, {
					Name: "gcsnopath1",
					GCS: &GCSSourceSpec{
						Type:     GCSArchive,
						Location: "blah",
					},
				}, {
					Name:       "gcswithpath",
					TargetPath: "gcs",
					GCS: &GCSSourceSpec{
						Type:     GCSArchive,
						Location: "blah",
					},
				}},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		desc: "source with duplicate names",
		build: &Build{
			Spec: BuildSpec{
				Sources: []SourceSpec{{
					Name: "sname",
					Git: &GitSourceSpec{
						Url:      "someurl",
						Revision: "revision",
					},
				}, {
					Name: "sname",
					Git: &GitSourceSpec{
						Url:      "someurl",
						Revision: "revision",
					},
				}},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
		reason: "sources with duplicate names",
	}, {
		desc: "a source with subpath",
		build: &Build{
			Spec: BuildSpec{
				Sources: []SourceSpec{{
					Name: "sname",
					Git: &GitSourceSpec{
						Url:      "someurl",
						Revision: "revision",
					},
					SubPath: "go",
				}},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
	}, {
		desc: "sources with subpath",
		build: &Build{
			Spec: BuildSpec{
				Sources: []SourceSpec{{
					Name: "sname",
					Git: &GitSourceSpec{
						Url:      "someurl",
						Revision: "revision",
					},
					SubPath: "go",
				}, {
					Name: "anothername",
					Git: &GitSourceSpec{
						Url:      "someurl",
						Revision: "revision",
					},
					SubPath: "ruby",
				}},
				Steps: []corev1.Container{{
					Name:  "foo",
					Image: "gcr.io/foo-bar/baz:latest",
				}},
			},
		},
		reason: "sources without subpaths",
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
