/*
Copyright 2019 The Kubernetes Authors.

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
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CloudEventsListenerSpec defines the desired state of CloudEventsListener
type CloudEventsListenerSpec struct {
	// Cloud event type we want to listen for
	// https://godoc.org/github.com/cloudevents/sdk-go/pkg/cloudevents#Event.Type
	CloudEventType string `json:"cloud-event-type"` // cloudevents.Event::GetType()
	// The repository location we are interested in.
	Repo string `json:"repo"`
	// Git branch we will create builds for.
	Branch string `json:"branch"`
	// Namespace where the build should be created.
	Namespace string `json:"namespace"`
	// Type of we are expecting - current support for github or gcs.
	SourceType string `json:"source-type"`
	// Status of the listener
	Status *CloudEventsListenerSpecStatus `json:"spec,omitempty"`
	// Build this listener will create
	Build *Build `json:"build,omitempty"`
}

// CloudEventsListenerSpecStatus is the status of the listener
type CloudEventsListenerSpecStatus string

// CloudEventsListenerStatus defines the observed state of CloudEventsListener
type CloudEventsListenerStatus struct {
	duckv1alpha1.Status `json:",inline"`
	Namespace           string `json:"namespace"`
	StatefulSetName     string `json:"statefulSetName"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudEventsListener is the Schema for the cloudeventslisteners API
// +k8s:openapi-gen=true
type CloudEventsListener struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudEventsListenerSpec   `json:"spec,omitempty"`
	Status CloudEventsListenerStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CloudEventsListenerList contains a list of CloudEventsListener
type CloudEventsListenerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudEventsListener `json:"items"`
}

// TemplateSpec provides access to the CloudEventsListener spec
func (c *CloudEventsListener) TemplateSpec() CloudEventsListenerSpec {
	return c.Spec
}
