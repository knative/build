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

// ClusterBuildTemplateLister helps list ClusterBuildTemplates.
type ClusterBuildTemplateLister interface {
	// List lists all ClusterBuildTemplates in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.ClusterBuildTemplate, err error)
	// ClusterBuildTemplates returns an object that can list and get ClusterBuildTemplates.
	ClusterBuildTemplates(namespace string) ClusterBuildTemplateNamespaceLister
	ClusterBuildTemplateListerExpansion
}

// clusterBuildTemplateLister implements the ClusterBuildTemplateLister interface.
type clusterBuildTemplateLister struct {
	indexer cache.Indexer
}

// NewClusterBuildTemplateLister returns a new ClusterBuildTemplateLister.
func NewClusterBuildTemplateLister(indexer cache.Indexer) ClusterBuildTemplateLister {
	return &clusterBuildTemplateLister{indexer: indexer}
}

// List lists all ClusterBuildTemplates in the indexer.
func (s *clusterBuildTemplateLister) List(selector labels.Selector) (ret []*v1alpha1.ClusterBuildTemplate, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ClusterBuildTemplate))
	})
	return ret, err
}

// ClusterBuildTemplates returns an object that can list and get ClusterBuildTemplates.
func (s *clusterBuildTemplateLister) ClusterBuildTemplates(namespace string) ClusterBuildTemplateNamespaceLister {
	return clusterBuildTemplateNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// ClusterBuildTemplateNamespaceLister helps list and get ClusterBuildTemplates.
type ClusterBuildTemplateNamespaceLister interface {
	// List lists all ClusterBuildTemplates in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.ClusterBuildTemplate, err error)
	// Get retrieves the ClusterBuildTemplate from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.ClusterBuildTemplate, error)
	ClusterBuildTemplateNamespaceListerExpansion
}

// clusterBuildTemplateNamespaceLister implements the ClusterBuildTemplateNamespaceLister
// interface.
type clusterBuildTemplateNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all ClusterBuildTemplates in the indexer for a given namespace.
func (s clusterBuildTemplateNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.ClusterBuildTemplate, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.ClusterBuildTemplate))
	})
	return ret, err
}

// Get retrieves the ClusterBuildTemplate from the indexer for a given namespace and name.
func (s clusterBuildTemplateNamespaceLister) Get(name string) (*v1alpha1.ClusterBuildTemplate, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("clusterbuildtemplate"), name)
	}
	return obj.(*v1alpha1.ClusterBuildTemplate), nil
}