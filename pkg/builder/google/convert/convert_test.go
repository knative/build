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
	"testing"

	"google.golang.org/api/cloudbuild/v1"

	v1alpha1 "github.com/google/build-crd/pkg/apis/cloudbuild/v1alpha1"
	"github.com/google/build-crd/pkg/buildtest"
)

func TestCloudbuildYAMLs(t *testing.T) {
	inputs := []string{
		"cloudbuilders/bazel/cloudbuild.yaml",
		"cloudbuilders/dotnet/cloudbuild.yaml",
		"cloudbuilders/gcloud/cloudbuild.yaml",
		"cloudbuilders/git/cloudbuild.yaml",
		"cloudbuilders/go/cloudbuild.yaml",
		"cloudbuilders/golang-project/cloudbuild.yaml",
		"cloudbuilders/gsutil/cloudbuild.yaml",
		"cloudbuilders/kubectl/cloudbuild.yaml",
		"cloudbuilders/wget/cloudbuild.yaml",
		"cloudbuilders/yarn/cloudbuild.yaml",
		"cloudbuilders/mvn/cloudbuild.yaml",
		"cloudbuilders/npm/cloudbuild.yaml",

		// These contain wait_for:
		//"cloudbuilders/docker/cloudbuild.yaml",
		//"cloudbuilders/gradle/cloudbuild.yaml",
		//"cloudbuilders/javac/cloudbuild.yaml",
	}

	for _, in := range inputs {
		var og cloudbuild.Build
		if err := buildtest.DataAs(in, &og); err != nil {
			t.Fatalf("Unexpected error in buildtest.DataAs(%q, cloudbuild.Build): %v", in, err)
		}
		// We don't support rountripping Images, so clear this field since the testdata we are using relies
		// on that instead of simply specifying docker push steps.
		og.Images = nil
		b, err := ToCRD(&og)
		if err != nil {
			t.Errorf("Unable to convert %q to CRD: %v", in, err)
		}
		bs, err := FromCRD(b)
		if err != nil {
			t.Errorf("Unable to convert %q from CRD: %v", in, err)
		}
		// Compare the pretty json because we don't care whether slice fields are empty or nil.
		// e.g. we want omitempty semantics.
		if ogjson, err := buildtest.PrettyJSON(og); err != nil {
			t.Errorf("Unexpected failure calling PrettyJSON(og=%v): %v", og, err)
		} else if bsjson, err := buildtest.PrettyJSON(bs); err != nil {
			t.Errorf("Unexpected failure calling PrettyJSON(bs=%v): %v", bs, err)
		} else if ogjson != bsjson {
			t.Errorf("Roundtrip(%q); want %v, got %v", in, ogjson, bsjson)
		}
	}
}

func TestSupportedCRDs(t *testing.T) {
	inputs := []string{
		"buildcrd/testdata/helloworld.yaml",
		"buildcrd/testdata/two-step.yaml",
		"buildcrd/testdata/env.yaml",
		"buildcrd/testdata/workingdir.yaml",

		// GCB doesn't model privilege.
		// NOTE: we cannot roundtrip this, but it should work as translated on GCB.
		// "buildcrd/testdata/security-context.yaml",

		// GCB doesn't support the resource stanza.
		// TODO(mattmoor): Perform a lossy translation to machine type?
		// "buildcrd/testdata/resources.yaml",

		"buildcrd/testdata/custom-source.yaml",

		"buildcrd/testdata/git-branch.yaml",
		"buildcrd/testdata/git-tag.yaml",
		"buildcrd/testdata/git-commit.yaml",
	}

	for _, in := range inputs {
		var og v1alpha1.BuildSpec
		if err := buildtest.DataAs(in, &og); err != nil {
			t.Fatalf("Unexpected error in buildtest.DataAs(%q, v1alpha1.BuildSpec): %v", in, err)
		}
		b, err := FromCRD(&og)
		if err != nil {
			t.Errorf("Unable to convert %q from CRD: %v", in, err)
		}
		bs, err := ToCRD(b)
		if err != nil {
			t.Errorf("Unable to convert %q to CRD: %v", in, err)
		}
		// Compare the pretty json because we don't care whether slice fields are empty or nil.
		// e.g. we want omitempty semantics.
		if ogjson, err := buildtest.PrettyJSON(og); err != nil {
			t.Errorf("Unexpected failure calling PrettyJSON(og=%v): %v", og, err)
		} else if bjson, err := buildtest.PrettyJSON(b); err != nil {
			t.Errorf("Unexpected failure calling PrettyJSON(b=%v): %v", b, err)
		} else if bsjson, err := buildtest.PrettyJSON(bs); err != nil {
			t.Errorf("Unexpected failure calling PrettyJSON(bs=%v): %v", bs, err)
		} else if ogjson != bsjson {
			t.Errorf("Roundtrip(%q); want %v, got %v; intermediate: %v", in, ogjson, bsjson, bjson)
		}
	}
}

func TestUnsupportedCRDs(t *testing.T) {
	inputs := []string{
		// Downward API is not supported.
		"buildcrd/testdata/env-valuefrom.yaml",

		// GCB doesn't support other volume types.
		"buildcrd/testdata/volumes.yaml",

		// GCB doesn't support refs.
		"buildcrd/testdata/git-ref.yaml",

		// GCB doesn't support any Git but CSR
		"buildcrd/testdata/git-branch-github.yaml",
	}

	for _, in := range inputs {
		var og v1alpha1.BuildSpec
		if err := buildtest.DataAs(in, &og); err != nil {
			t.Fatalf("Unexpected error in buildtest.DataAs(%q, v1alpha1.BuildSpec): %v", in, err)
		}

		if bs, err := FromCRD(&og); err == nil {
			t.Errorf("FromCRD(%v); wanted error, but got: %v", og, bs)
		}
	}
}

func TestSupportedCRDsOneWay(t *testing.T) {
	inputs := []string{
		"buildcrd/testdata/docker-build.yaml",
	}

	for _, in := range inputs {
		var og v1alpha1.BuildSpec
		if err := buildtest.DataAs(in, &og); err != nil {
			t.Fatalf("Unexpected error in buildtest.DataAs(%q, v1alpha1.BuildSpec): %v", in, err)
		}
		b, err := FromCRD(&og)
		if err != nil {
			t.Errorf("Unable to convert %q from CRD: %v", in, err)
		}
		bs, err := ToCRD(b)
		if err != nil {
			t.Errorf("Unable to convert %q to CRD: %v", in, err)
		}
		// Compare the pretty json because we don't care whether slice fields are empty or nil.
		// e.g. we want omitempty semantics.
		if ogjson, err := buildtest.PrettyJSON(og); err != nil {
			t.Errorf("Unexpected failure calling PrettyJSON(og=%v): %v", og, err)
		} else if bjson, err := buildtest.PrettyJSON(b); err != nil {
			t.Errorf("Unexpected failure calling PrettyJSON(b=%v): %v", b, err)
		} else if bsjson, err := buildtest.PrettyJSON(bs); err != nil {
			t.Errorf("Unexpected failure calling PrettyJSON(bs=%v): %v", bs, err)
		} else if ogjson == bsjson {
			t.Errorf("Roundtrip(%q); want different, got same: %v; intermediate: %v", in, bsjson, bjson)
		}
	}
}
