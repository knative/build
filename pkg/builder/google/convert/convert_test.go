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
	"testing"

	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/knative/build/pkg/buildtest"
	cloudbuild "google.golang.org/api/cloudbuild/v1"
)

func TestCloudbuildYAMLs(t *testing.T) {
	inputs := []string{
		"testdata/cloudbuilders/bazel/cloudbuild.yaml",
		"testdata/cloudbuilders/dotnet/cloudbuild.yaml",
		"testdata/cloudbuilders/gcloud/cloudbuild.yaml",
		"testdata/cloudbuilders/git/cloudbuild.yaml",
		"testdata/cloudbuilders/go/cloudbuild.yaml",
		"testdata/cloudbuilders/golang-project/cloudbuild.yaml",
		"testdata/cloudbuilders/gsutil/cloudbuild.yaml",
		"testdata/cloudbuilders/kubectl/cloudbuild.yaml",
		"testdata/cloudbuilders/wget/cloudbuild.yaml",
		"testdata/cloudbuilders/yarn/cloudbuild.yaml",
		"testdata/cloudbuilders/mvn/cloudbuild.yaml",
		"testdata/cloudbuilders/npm/cloudbuild.yaml",

		// These contain wait_for:
		//"testdata/cloudbuilders/docker/cloudbuild.yaml",
		//"testdata/cloudbuilders/gradle/cloudbuild.yaml",
		//"testdata/cloudbuilders/javac/cloudbuild.yaml",
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
		"testdata/helloworld.yaml",
		"testdata/two-step.yaml",
		"testdata/env.yaml",
		"testdata/workingdir.yaml",

		// GCB doesn't model privilege.
		// NOTE: we cannot roundtrip this, but it should work as translated on GCB.
		// "testdata/security-context.yaml",

		// GCB doesn't support the resource stanza.
		// TODO(mattmoor): Perform a lossy translation to machine type?
		// "testdata/resources.yaml",

		"testdata/custom-source.yaml",

		"testdata/gcs-archive.yaml",
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
		"testdata/env-valuefrom.yaml",

		// GCB doesn't support other volume types.
		"testdata/volumes.yaml",

		// GCB doesn't support arbitrary Git refs.
		"testdata/git-revision.yaml",

		// GCB doesn't support source manifests.
		"testdata/gcs-manifest.yaml",
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
		"testdata/docker-build.yaml",
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
