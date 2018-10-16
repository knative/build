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

package builder

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
)

func TestIsDone(t *testing.T) {
	testcases := []struct {
		status *v1alpha1.BuildStatus
		done   bool
	}{
		{
			status: nil,
			done:   false,
		},
		{
			status: &v1alpha1.BuildStatus{},
			done:   false,
		},
		{
			status: &v1alpha1.BuildStatus{
				Conditions: duckv1alpha1.Conditions{{
					Type: "Pending",
				}},
			},
			done: false,
		},
		{
			status: &v1alpha1.BuildStatus{
				Conditions: duckv1alpha1.Conditions{{
					Type:   v1alpha1.BuildSucceeded,
					Status: corev1.ConditionUnknown,
				}},
			},
			done: false,
		},
		{
			status: &v1alpha1.BuildStatus{
				Conditions: duckv1alpha1.Conditions{{
					Type: v1alpha1.BuildSucceeded,
				}},
			},
			done: true,
		},
	}
	for _, tc := range testcases {
		if d := IsDone(tc.status); d != tc.done {
			t.Fatalf("status %+v, expected %v, got %v", tc.status, tc.done, d)
		}
	}
}

func TestIsTimeout(t *testing.T) {
	testcases := []struct {
		status       *v1alpha1.BuildStatus
		buildTimeout *metav1.Duration
		timeout      bool
	}{
		{
			status:  nil,
			timeout: false,
		},
		{
			status: &v1alpha1.BuildStatus{
				Conditions: duckv1alpha1.Conditions{{
					Type: "Pending",
				}},
			},
			timeout: false,
		},
		{
			status: &v1alpha1.BuildStatus{
				Conditions: duckv1alpha1.Conditions{{
					Type: "Pending",
				}},
				StartTime: metav1.Now(),
			},
			timeout: false,
		},
		{
			status: &v1alpha1.BuildStatus{
				Conditions: duckv1alpha1.Conditions{{
					Type: "Pending",
				}},
				StartTime: metav1.Time{
					Time: time.Now().Add(-5 * time.Minute),
				},
			},
			timeout: false,
		},
		{
			status: &v1alpha1.BuildStatus{
				Conditions: duckv1alpha1.Conditions{{
					Type: "Pending",
				}},
				StartTime: metav1.Time{
					Time: time.Now().Add(-11 * time.Minute),
				},
			},
			buildTimeout: &metav1.Duration{Duration: 20 * time.Minute},
			timeout:      false,
		},
		{
			status: &v1alpha1.BuildStatus{
				Conditions: duckv1alpha1.Conditions{{
					Type: "Pending",
				}},
				StartTime: metav1.Time{
					Time: time.Now().Add(-11 * time.Minute),
				},
			},
			buildTimeout: &metav1.Duration{Duration: 9 * time.Minute},
			timeout:      true,
		},
	}
	for _, tc := range testcases {
		if b := IsTimeout(tc.status, tc.buildTimeout); b != tc.timeout {
			t.Fatalf("status %+v, expected %v, got %v", tc.status, tc.timeout, b)
		}
	}
}

func TestErrorMessage(t *testing.T) {
	testcases := []struct {
		status  *v1alpha1.BuildStatus
		message string
		error   bool
	}{
		{
			status:  nil,
			error:   false,
			message: "",
		},
		{
			status:  &v1alpha1.BuildStatus{},
			error:   false,
			message: "",
		},
		{
			status: &v1alpha1.BuildStatus{
				Conditions: duckv1alpha1.Conditions{{
					Type: "Pending",
				}},
			},
			error:   false,
			message: "",
		},
		{
			status: &v1alpha1.BuildStatus{
				Conditions: duckv1alpha1.Conditions{{
					Type:   v1alpha1.BuildSucceeded,
					Status: corev1.ConditionUnknown,
				}},
			},
			error:   false,
			message: "",
		},
		{
			status: &v1alpha1.BuildStatus{
				Conditions: duckv1alpha1.Conditions{{
					Type:    v1alpha1.BuildSucceeded,
					Status:  corev1.ConditionFalse,
					Message: "Error building",
				}},
			},
			error:   true,
			message: "Error building",
		},
	}
	for _, tc := range testcases {
		if m, e := ErrorMessage(tc.status); e != tc.error || m != tc.message {
			t.Fatalf("status %+v, expected (%s,%v), got (%s,%v)", tc.status, tc.message, tc.error, m, e)
		}
	}
}
