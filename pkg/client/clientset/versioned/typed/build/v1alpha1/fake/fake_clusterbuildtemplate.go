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

// FakeClusterBuildTemplates implements ClusterBuildTemplateInterface
type FakeClusterBuildTemplates struct {
	Fake *FakeBuildV1alpha1
	ns   string
}

var clusterbuildtemplatesResource = schema.GroupVersionResource{Group: "build.knative.dev", Version: "v1alpha1", Resource: "clusterbuildtemplates"}

var clusterbuildtemplatesKind = schema.GroupVersionKind{Group: "build.knative.dev", Version: "v1alpha1", Kind: "ClusterBuildTemplate"}

// Get takes name of the clusterBuildTemplate, and returns the corresponding clusterBuildTemplate object, and an error if there is any.
func (c *FakeClusterBuildTemplates) Get(name string, options v1.GetOptions) (result *v1alpha1.ClusterBuildTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(clusterbuildtemplatesResource, c.ns, name), &v1alpha1.ClusterBuildTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterBuildTemplate), err
}

// List takes label and field selectors, and returns the list of ClusterBuildTemplates that match those selectors.
func (c *FakeClusterBuildTemplates) List(opts v1.ListOptions) (result *v1alpha1.ClusterBuildTemplateList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(clusterbuildtemplatesResource, clusterbuildtemplatesKind, c.ns, opts), &v1alpha1.ClusterBuildTemplateList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ClusterBuildTemplateList{ListMeta: obj.(*v1alpha1.ClusterBuildTemplateList).ListMeta}
	for _, item := range obj.(*v1alpha1.ClusterBuildTemplateList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested clusterBuildTemplates.
func (c *FakeClusterBuildTemplates) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(clusterbuildtemplatesResource, c.ns, opts))

}

// Create takes the representation of a clusterBuildTemplate and creates it.  Returns the server's representation of the clusterBuildTemplate, and an error, if there is any.
func (c *FakeClusterBuildTemplates) Create(clusterBuildTemplate *v1alpha1.ClusterBuildTemplate) (result *v1alpha1.ClusterBuildTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(clusterbuildtemplatesResource, c.ns, clusterBuildTemplate), &v1alpha1.ClusterBuildTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterBuildTemplate), err
}

// Update takes the representation of a clusterBuildTemplate and updates it. Returns the server's representation of the clusterBuildTemplate, and an error, if there is any.
func (c *FakeClusterBuildTemplates) Update(clusterBuildTemplate *v1alpha1.ClusterBuildTemplate) (result *v1alpha1.ClusterBuildTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(clusterbuildtemplatesResource, c.ns, clusterBuildTemplate), &v1alpha1.ClusterBuildTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterBuildTemplate), err
}

// Delete takes name of the clusterBuildTemplate and deletes it. Returns an error if one occurs.
func (c *FakeClusterBuildTemplates) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(clusterbuildtemplatesResource, c.ns, name), &v1alpha1.ClusterBuildTemplate{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeClusterBuildTemplates) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(clusterbuildtemplatesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.ClusterBuildTemplateList{})
	return err
}

// Patch applies the patch and returns the patched clusterBuildTemplate.
func (c *FakeClusterBuildTemplates) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.ClusterBuildTemplate, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(clusterbuildtemplatesResource, c.ns, name, data, subresources...), &v1alpha1.ClusterBuildTemplate{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ClusterBuildTemplate), err
}