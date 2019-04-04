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
package v1alpha1

import (
	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	scheme "github.com/knative/build/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// CloudEventsListenersGetter has a method to return a CloudEventsListenerInterface.
// A group's client should implement this interface.
type CloudEventsListenersGetter interface {
	CloudEventsListeners(namespace string) CloudEventsListenerInterface
}

// CloudEventsListenerInterface has methods to work with CloudEventsListener resources.
type CloudEventsListenerInterface interface {
	Create(*v1alpha1.CloudEventsListener) (*v1alpha1.CloudEventsListener, error)
	Update(*v1alpha1.CloudEventsListener) (*v1alpha1.CloudEventsListener, error)
	UpdateStatus(*v1alpha1.CloudEventsListener) (*v1alpha1.CloudEventsListener, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.CloudEventsListener, error)
	List(opts v1.ListOptions) (*v1alpha1.CloudEventsListenerList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.CloudEventsListener, err error)
	CloudEventsListenerExpansion
}

// cloudEventsListeners implements CloudEventsListenerInterface
type cloudEventsListeners struct {
	client rest.Interface
	ns     string
}

// newCloudEventsListeners returns a CloudEventsListeners
func newCloudEventsListeners(c *BuildV1alpha1Client, namespace string) *cloudEventsListeners {
	return &cloudEventsListeners{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the cloudEventsListener, and returns the corresponding cloudEventsListener object, and an error if there is any.
func (c *cloudEventsListeners) Get(name string, options v1.GetOptions) (result *v1alpha1.CloudEventsListener, err error) {
	result = &v1alpha1.CloudEventsListener{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("cloudeventslisteners").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of CloudEventsListeners that match those selectors.
func (c *cloudEventsListeners) List(opts v1.ListOptions) (result *v1alpha1.CloudEventsListenerList, err error) {
	result = &v1alpha1.CloudEventsListenerList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("cloudeventslisteners").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested cloudEventsListeners.
func (c *cloudEventsListeners) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("cloudeventslisteners").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a cloudEventsListener and creates it.  Returns the server's representation of the cloudEventsListener, and an error, if there is any.
func (c *cloudEventsListeners) Create(cloudEventsListener *v1alpha1.CloudEventsListener) (result *v1alpha1.CloudEventsListener, err error) {
	result = &v1alpha1.CloudEventsListener{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("cloudeventslisteners").
		Body(cloudEventsListener).
		Do().
		Into(result)
	return
}

// Update takes the representation of a cloudEventsListener and updates it. Returns the server's representation of the cloudEventsListener, and an error, if there is any.
func (c *cloudEventsListeners) Update(cloudEventsListener *v1alpha1.CloudEventsListener) (result *v1alpha1.CloudEventsListener, err error) {
	result = &v1alpha1.CloudEventsListener{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("cloudeventslisteners").
		Name(cloudEventsListener.Name).
		Body(cloudEventsListener).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *cloudEventsListeners) UpdateStatus(cloudEventsListener *v1alpha1.CloudEventsListener) (result *v1alpha1.CloudEventsListener, err error) {
	result = &v1alpha1.CloudEventsListener{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("cloudeventslisteners").
		Name(cloudEventsListener.Name).
		SubResource("status").
		Body(cloudEventsListener).
		Do().
		Into(result)
	return
}

// Delete takes name of the cloudEventsListener and deletes it. Returns an error if one occurs.
func (c *cloudEventsListeners) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("cloudeventslisteners").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *cloudEventsListeners) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("cloudeventslisteners").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched cloudEventsListener.
func (c *cloudEventsListeners) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.CloudEventsListener, err error) {
	result = &v1alpha1.CloudEventsListener{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("cloudeventslisteners").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
