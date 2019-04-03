// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1 "kmodules.xyz/openshift/apis/security/v1"
)

// RangeAllocationLister helps list RangeAllocations.
type RangeAllocationLister interface {
	// List lists all RangeAllocations in the indexer.
	List(selector labels.Selector) (ret []*v1.RangeAllocation, err error)
	// Get retrieves the RangeAllocation from the index for a given name.
	Get(name string) (*v1.RangeAllocation, error)
	RangeAllocationListerExpansion
}

// rangeAllocationLister implements the RangeAllocationLister interface.
type rangeAllocationLister struct {
	indexer cache.Indexer
}

// NewRangeAllocationLister returns a new RangeAllocationLister.
func NewRangeAllocationLister(indexer cache.Indexer) RangeAllocationLister {
	return &rangeAllocationLister{indexer: indexer}
}

// List lists all RangeAllocations in the indexer.
func (s *rangeAllocationLister) List(selector labels.Selector) (ret []*v1.RangeAllocation, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.RangeAllocation))
	})
	return ret, err
}

// Get retrieves the RangeAllocation from the index for a given name.
func (s *rangeAllocationLister) Get(name string) (*v1.RangeAllocation, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("rangeallocation"), name)
	}
	return obj.(*v1.RangeAllocation), nil
}