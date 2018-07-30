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

	corev1 "k8s.io/api/core/v1"
)

func TestGoodString(t *testing.T) {
	ev, err := toEnvVarFromString("key=value")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if ev.Name != "key" {
		t.Errorf("Unexpected name; want \"key\", but got: %q", ev.Name)
	}
	if ev.Value != "value" {
		t.Errorf("Unexpected value; want \"value\", but got: %q", ev.Value)
	}
}

func TestComplexGoodString(t *testing.T) {
	ev, err := toEnvVarFromString("key=value=another-value")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if ev.Name != "key" {
		t.Errorf("Unexpected name; want \"key\", but got: %q", ev.Name)
	}
	if ev.Value != "value=another-value" {
		t.Errorf("Unexpected value; want \"value=another-value\", but got: %q", ev.Value)
	}
}

func TestBadString(t *testing.T) {
	ev, err := toEnvVarFromString("asdf")
	if err == nil {
		t.Errorf("toEnvVarFromString(asdf); wanted error, got: %v", ev)
	}
	// Make sure the list variety fails too.
	evs, err := toEnvFromAssociativeList([]string{"asdf"})
	if err == nil {
		t.Errorf("toEnvFromAssociativeList(asdf); wanted error, got: %v", evs)
	}
}

func TestDownwardAPI(t *testing.T) {
	ev := corev1.EnvVar{
		Name: "MY_NAMESPACE",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.namespace",
			},
		},
	}
	s, err := toStringFromEnvVar(&ev)
	if err == nil {
		t.Errorf("toEnvVarFromString(%v); wanted error, got: %v", ev, s)
	}
	// Make sure the list variety fails too.
	al, err := toAssociativeListFromEnv([]corev1.EnvVar{ev})
	if err == nil {
		t.Errorf("toAssociativeListFromEnv(%v); wanted error, got: %v", ev, al)
	}
}

func TestEnvRoundtrip(t *testing.T) {
	inputs := []string{
		"FOO=bar",
		"BAZ=blah",
		"PATH=/usr/bin:/bin:/user/me/bin",
	}
	ev, err := toEnvFromAssociativeList(inputs)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	result, err := toAssociativeListFromEnv(ev)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(inputs, result) {
		t.Errorf("Bad roundtrip; wanted %v, but got: %v", inputs, result)
	}
}
