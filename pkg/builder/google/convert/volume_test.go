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
	"reflect"
	"testing"

	"google.golang.org/api/cloudbuild/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestBadVolumeMounts(t *testing.T) {
	mpm := corev1.MountPropagationBidirectional
	badVolumeMounts := []corev1.VolumeMount{
		// We do not (currently) have a way of asking for a ReadOnly volume.
		{
			Name:      "foo",
			MountPath: "/bar",
			ReadOnly:  true,
		},
		// We do not (currently) have a way of asking for a subpath to be mounted.
		{
			Name:      "foo",
			MountPath: "/bar",
			SubPath:   "baz/blah",
		},
		// We do not (currently) have a way of asking for mount propagation.
		{
			Name:             "foo",
			MountPath:        "/bar",
			MountPropagation: &mpm,
		},
	}
	for _, vm := range badVolumeMounts {
		v, err := ToVolumeFromVolumeMount(&vm)
		if err == nil {
			t.Errorf("ToVolumeFromVolumeMount(%v); wanted error, got %v", vm, v)
		}
		// Make sure the list variety fails too.
		vs, err := ToVolumesFromVolumeMounts([]corev1.VolumeMount{vm})
		if err == nil {
			t.Errorf("ToVolumesFromVolumeMounts(%v); wanted error, got %v", vm, vs)
		}
	}
}

func TestVolumeRoundtrip(t *testing.T) {
	inputs := []*cloudbuild.Volume{
		{
			Name: "foo",
			Path: "/path1",
		},
		{
			Name: "bar",
			Path: "/path/2/here",
		},
		{
			Name: "baz",
			Path: "/another/path/to/somewhere",
		},
	}
	vms, err := ToVolumeMountsFromVolumes(inputs)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	result, err := ToVolumesFromVolumeMounts(vms)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(inputs, result) {
		t.Errorf("Bad roundtrip; wanted %v, but got: %v", inputs, result)
	}
}
