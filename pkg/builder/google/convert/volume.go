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
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"google.golang.org/api/cloudbuild/v1"

	"github.com/google/build-crd/pkg/builder"
)

func ToVolumeMountFromVolume(og *cloudbuild.Volume) (*corev1.VolumeMount, error) {
	return &corev1.VolumeMount{
		Name:      og.Name,
		MountPath: og.Path,
	}, nil
}

func ToVolumeFromVolumeMount(og *corev1.VolumeMount) (*cloudbuild.Volume, error) {
	if og.ReadOnly {
		return nil, &builder.ValidationError{
			Reason:  "ReadOnly",
			Message: fmt.Sprintf("container builder does not support ReadOnly volumes, got: %v", og),
		}
	}
	if og.MountPropagation != nil {
		return nil, &builder.ValidationError{
			Reason:  "MountPropagation",
			Message: fmt.Sprintf("container builder does not support mountPropagation on volumes, got: %v", og),
		}
	}
	if og.SubPath != "" {
		return nil, &builder.ValidationError{
			Reason:  "VolumeSubpath",
			Message: fmt.Sprintf("container builder does not support subPath on volumes, got: %v", og),
		}
	}

	return &cloudbuild.Volume{
		Name: og.Name,
		Path: og.MountPath,
	}, nil
}

func ToVolumeMountsFromVolumes(og []*cloudbuild.Volume) ([]corev1.VolumeMount, error) {
	al := make([]corev1.VolumeMount, 0, len(og))
	for _, v := range og {
		vm, err := ToVolumeMountFromVolume(v)
		if err != nil {
			return nil, err
		}
		al = append(al, *vm)
	}
	return al, nil
}

func ToVolumesFromVolumeMounts(og []corev1.VolumeMount) ([]*cloudbuild.Volume, error) {
	al := make([]*cloudbuild.Volume, 0, len(og))
	for _, vm := range og {
		v, err := ToVolumeFromVolumeMount(&vm)
		if err != nil {
			return nil, err
		}
		al = append(al, v)
	}
	return al, nil
}
