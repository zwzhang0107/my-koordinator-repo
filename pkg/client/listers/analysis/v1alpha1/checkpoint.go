/*
Copyright 2022 The Koordinator Authors.

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

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/koordinator-sh/koordinator/apis/analysis/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// CheckpointLister helps list Checkpoints.
// All objects returned here must be treated as read-only.
type CheckpointLister interface {
	// List lists all Checkpoints in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Checkpoint, err error)
	// Checkpoints returns an object that can list and get Checkpoints.
	Checkpoints(namespace string) CheckpointNamespaceLister
	CheckpointListerExpansion
}

// checkpointLister implements the CheckpointLister interface.
type checkpointLister struct {
	indexer cache.Indexer
}

// NewCheckpointLister returns a new CheckpointLister.
func NewCheckpointLister(indexer cache.Indexer) CheckpointLister {
	return &checkpointLister{indexer: indexer}
}

// List lists all Checkpoints in the indexer.
func (s *checkpointLister) List(selector labels.Selector) (ret []*v1alpha1.Checkpoint, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Checkpoint))
	})
	return ret, err
}

// Checkpoints returns an object that can list and get Checkpoints.
func (s *checkpointLister) Checkpoints(namespace string) CheckpointNamespaceLister {
	return checkpointNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// CheckpointNamespaceLister helps list and get Checkpoints.
// All objects returned here must be treated as read-only.
type CheckpointNamespaceLister interface {
	// List lists all Checkpoints in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.Checkpoint, err error)
	// Get retrieves the Checkpoint from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.Checkpoint, error)
	CheckpointNamespaceListerExpansion
}

// checkpointNamespaceLister implements the CheckpointNamespaceLister
// interface.
type checkpointNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all Checkpoints in the indexer for a given namespace.
func (s checkpointNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.Checkpoint, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.Checkpoint))
	})
	return ret, err
}

// Get retrieves the Checkpoint from the indexer for a given namespace and name.
func (s checkpointNamespaceLister) Get(name string) (*v1alpha1.Checkpoint, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("checkpoint"), name)
	}
	return obj.(*v1alpha1.Checkpoint), nil
}
