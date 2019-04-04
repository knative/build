/*
Copyright 2019 The Knative Authors

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
package fake

import (
	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeCloudEventsListeners implements CloudEventsListenerInterface
type FakeCloudEventsListeners struct {
	Fake *FakeBuildV1alpha1
	ns   string
}

var cloudeventslistenersResource = schema.GroupVersionResource{Group: "build.knative.dev", Version: "v1alpha1", Resource: "cloudeventslisteners"}

var cloudeventslistenersKind = schema.GroupVersionKind{Group: "build.knative.dev", Version: "v1alpha1", Kind: "CloudEventsListener"}

// Get takes name of the cloudEventsListener, and returns the corresponding cloudEventsListener object, and an error if there is any.
func (c *FakeCloudEventsListeners) Get(name string, options v1.GetOptions) (result *v1alpha1.CloudEventsListener, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(cloudeventslistenersResource, c.ns, name), &v1alpha1.CloudEventsListener{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.CloudEventsListener), err
}

// List takes label and field selectors, and returns the list of CloudEventsListeners that match those selectors.
func (c *FakeCloudEventsListeners) List(opts v1.ListOptions) (result *v1alpha1.CloudEventsListenerList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(cloudeventslistenersResource, cloudeventslistenersKind, c.ns, opts), &v1alpha1.CloudEventsListenerList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.CloudEventsListenerList{ListMeta: obj.(*v1alpha1.CloudEventsListenerList).ListMeta}
	for _, item := range obj.(*v1alpha1.CloudEventsListenerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested cloudEventsListeners.
func (c *FakeCloudEventsListeners) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(cloudeventslistenersResource, c.ns, opts))

}

// Create takes the representation of a cloudEventsListener and creates it.  Returns the server's representation of the cloudEventsListener, and an error, if there is any.
func (c *FakeCloudEventsListeners) Create(cloudEventsListener *v1alpha1.CloudEventsListener) (result *v1alpha1.CloudEventsListener, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(cloudeventslistenersResource, c.ns, cloudEventsListener), &v1alpha1.CloudEventsListener{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.CloudEventsListener), err
}

// Update takes the representation of a cloudEventsListener and updates it. Returns the server's representation of the cloudEventsListener, and an error, if there is any.
func (c *FakeCloudEventsListeners) Update(cloudEventsListener *v1alpha1.CloudEventsListener) (result *v1alpha1.CloudEventsListener, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(cloudeventslistenersResource, c.ns, cloudEventsListener), &v1alpha1.CloudEventsListener{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.CloudEventsListener), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeCloudEventsListeners) UpdateStatus(cloudEventsListener *v1alpha1.CloudEventsListener) (*v1alpha1.CloudEventsListener, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(cloudeventslistenersResource, "status", c.ns, cloudEventsListener), &v1alpha1.CloudEventsListener{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.CloudEventsListener), err
}

// Delete takes name of the cloudEventsListener and deletes it. Returns an error if one occurs.
func (c *FakeCloudEventsListeners) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(cloudeventslistenersResource, c.ns, name), &v1alpha1.CloudEventsListener{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeCloudEventsListeners) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(cloudeventslistenersResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.CloudEventsListenerList{})
	return err
}

// Patch applies the patch and returns the patched cloudEventsListener.
func (c *FakeCloudEventsListeners) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.CloudEventsListener, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(cloudeventslistenersResource, c.ns, name, data, subresources...), &v1alpha1.CloudEventsListener{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.CloudEventsListener), err
}
