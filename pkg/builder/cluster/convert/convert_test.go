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

package convert

import (
	"github.com/elafros/build-crd/pkg/buildtest"

	v1alpha1 "github.com/elafros/build-crd/pkg/apis/build/v1alpha1"
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
		"buildcrd/testdata/helloworld.yaml",
		"buildcrd/testdata/two-step.yaml",
		"buildcrd/testdata/env.yaml",
		"buildcrd/testdata/env-valuefrom.yaml",
		"buildcrd/testdata/workingdir.yaml",
		"buildcrd/testdata/resources.yaml",
		"buildcrd/testdata/security-context.yaml",
		"buildcrd/testdata/volumes.yaml",
		"buildcrd/testdata/custom-source.yaml",

		"buildcrd/testdata/git-branch.yaml",
		"buildcrd/testdata/git-branch-github.yaml",
		"buildcrd/testdata/git-tag.yaml",
		"buildcrd/testdata/git-commit.yaml",
		"buildcrd/testdata/git-ref.yaml",
	}

	for _, in := range inputs {
		og, err := read2CRD(in)
		if err != nil {
			t.Fatalf("Unexpected error in read2CRD(%q): %v", in, err)
		}
		cs := fakek8s.NewSimpleClientset(&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "default"}})
		j, err := FromCRD(og, cs)
		if err != nil {
			t.Errorf("Unable to convert %q from CRD: %v", in, err)
			continue
		}
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
