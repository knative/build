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

package convert

import (
	"github.com/knative/build/pkg/buildtest"

	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakek8s "k8s.io/client-go/kubernetes/fake"

	"testing"
)

func read2CRD(f string) (*v1alpha1.Build, error) {
	var bs v1alpha1.Build
	if err := buildtest.DataAs(f, &bs.Spec); err != nil {
		return nil, err
	}
	return &bs, nil
}

func TestParsing(t *testing.T) {
	inputs := []string{
		"testdata/helloworld.yaml",
		"testdata/two-step.yaml",
		"testdata/env.yaml",
		"testdata/env-valuefrom.yaml",
		"testdata/workingdir.yaml",
		"testdata/resources.yaml",
		"testdata/security-context.yaml",
		"testdata/volumes.yaml",
		"testdata/custom-source.yaml",

		"testdata/git-revision.yaml",

		"testdata/gcs-archive.yaml",
		"testdata/gcs-manifest.yaml",
	}

	for _, in := range inputs {
		og, err := read2CRD(in)
		if err != nil {
			t.Fatalf("Unexpected error in read2CRD(%q): %v", in, err)
		}
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{Name: "default"},
			Secrets: []corev1.ObjectReference{
				{
					Name: "multi-creds",
				},
			},
		}
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "multi-creds",
				Annotations: map[string]string{"build.dev/docker-0": "https://us.gcr.io",
					"build.dev/docker-1": "https://docker.io",
					"build.dev/git-0":    "github.com",
					"build.dev/git-1":    "gitlab.com",
				}},
			Type: "kubernetes.io/basic-auth",
			Data: map[string][]byte{
				"username": []byte("foo"),
				"password": []byte("BestEver"),
			},
		}
		cs := fakek8s.NewSimpleClientset(sa, secret)
		j, err := FromCRD(og, cs)
		if err != nil {
			t.Errorf("Unable to convert %q from CRD: %v", in, err)
			continue
		}

		// Verify that secrets are loaded correctly.
		if j.Spec.ServiceAccountName != "" {
			for _, vol := range j.Spec.Volumes {
				if vol.Name == "secret-volume-multi-creds" {
					if vol.Secret.SecretName != "multi-creds" {
						t.Errorf("Expected multi-creds to be mounted in Pod %v", j.Spec)
					}
				}
			}
			expected := map[string]int{"https://us.gcr.io": 1, "https://docker.io": 1, "github.com": 1, "gitlab.com": 1}
			for _, a := range j.Spec.InitContainers[0].Args {
				expected[a] -= 1
			}
			for k, c := range expected {
				if c > 0 {
					t.Errorf("Expected arg related to %s in args, got %v", k, j.Spec.InitContainers[0].Args)
				}
			}
		}
		// Verify that reverse transformation works.
		
		b, err := ToCRD(j)
		if err != nil {
			t.Errorf("Unable to convert %q to CRD: %v", in, err)
			continue
		}
		// Compare the pretty json because we don't care whether slice fields are empty or nil.
		// e.g. we want omitempty semantics.
		if ogjson, err := buildtest.PrettyJSON(og); err != nil {
			t.Errorf("Unexpected failure calling PrettyJSON(og=%v): %v", og, err)
		} else if bjson, err := buildtest.PrettyJSON(b); err != nil {
			t.Errorf("Unexpected failure calling PrettyJSON(b=%v): %v", b, err)
		} else if ogjson != bjson {
			t.Errorf("Roundtrip(%q); want %v, got %v", in, ogjson, bjson)
		}
	}
}
