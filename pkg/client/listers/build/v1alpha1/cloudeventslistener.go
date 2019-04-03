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
package v1alpha1

import (
	v1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// CloudEventsListenerLister helps list CloudEventsListeners.
type CloudEventsListenerLister interface {
	// List lists all CloudEventsListeners in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.CloudEventsListener, err error)
	// CloudEventsListeners returns an object that can list and get CloudEventsListeners.
	CloudEventsListeners(namespace string) CloudEventsListenerNamespaceLister
	CloudEventsListenerListerExpansion
}

// cloudEventsListenerLister implements the CloudEventsListenerLister interface.
type cloudEventsListenerLister struct {
	indexer cache.Indexer
}

// NewCloudEventsListenerLister returns a new CloudEventsListenerLister.
func NewCloudEventsListenerLister(indexer cache.Indexer) CloudEventsListenerLister {
	return &cloudEventsListenerLister{indexer: indexer}
}

// List lists all CloudEventsListeners in the indexer.
func (s *cloudEventsListenerLister) List(selector labels.Selector) (ret []*v1alpha1.CloudEventsListener, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.CloudEventsListener))
	})
	return ret, err
}

// CloudEventsListeners returns an object that can list and get CloudEventsListeners.
func (s *cloudEventsListenerLister) CloudEventsListeners(namespace string) CloudEventsListenerNamespaceLister {
	return cloudEventsListenerNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// CloudEventsListenerNamespaceLister helps list and get CloudEventsListeners.
type CloudEventsListenerNamespaceLister interface {
	// List lists all CloudEventsListeners in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.CloudEventsListener, err error)
	// Get retrieves the CloudEventsListener from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.CloudEventsListener, error)
	CloudEventsListenerNamespaceListerExpansion
}

// cloudEventsListenerNamespaceLister implements the CloudEventsListenerNamespaceLister
// interface.
type cloudEventsListenerNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all CloudEventsListeners in the indexer for a given namespace.
func (s cloudEventsListenerNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.CloudEventsListener, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.CloudEventsListener))
	})
	return ret, err
}

// Get retrieves the CloudEventsListener from the indexer for a given namespace and name.
func (s cloudEventsListenerNamespaceLister) Get(name string) (*v1alpha1.CloudEventsListener, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("cloudeventslistener"), name)
	}
	return obj.(*v1alpha1.CloudEventsListener), nil
}
