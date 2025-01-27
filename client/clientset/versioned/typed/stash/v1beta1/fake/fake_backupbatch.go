/*
Copyright 2019 The Stash Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
	v1beta1 "stash.appscode.dev/stash/apis/stash/v1beta1"
)

// FakeBackupBatches implements BackupBatchInterface
type FakeBackupBatches struct {
	Fake *FakeStashV1beta1
	ns   string
}

var backupbatchesResource = schema.GroupVersionResource{Group: "stash.appscode.com", Version: "v1beta1", Resource: "backupbatches"}

var backupbatchesKind = schema.GroupVersionKind{Group: "stash.appscode.com", Version: "v1beta1", Kind: "BackupBatch"}

// Get takes name of the backupBatch, and returns the corresponding backupBatch object, and an error if there is any.
func (c *FakeBackupBatches) Get(name string, options v1.GetOptions) (result *v1beta1.BackupBatch, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(backupbatchesResource, c.ns, name), &v1beta1.BackupBatch{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.BackupBatch), err
}

// List takes label and field selectors, and returns the list of BackupBatches that match those selectors.
func (c *FakeBackupBatches) List(opts v1.ListOptions) (result *v1beta1.BackupBatchList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(backupbatchesResource, backupbatchesKind, c.ns, opts), &v1beta1.BackupBatchList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1beta1.BackupBatchList{ListMeta: obj.(*v1beta1.BackupBatchList).ListMeta}
	for _, item := range obj.(*v1beta1.BackupBatchList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested backupBatches.
func (c *FakeBackupBatches) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(backupbatchesResource, c.ns, opts))

}

// Create takes the representation of a backupBatch and creates it.  Returns the server's representation of the backupBatch, and an error, if there is any.
func (c *FakeBackupBatches) Create(backupBatch *v1beta1.BackupBatch) (result *v1beta1.BackupBatch, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(backupbatchesResource, c.ns, backupBatch), &v1beta1.BackupBatch{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.BackupBatch), err
}

// Update takes the representation of a backupBatch and updates it. Returns the server's representation of the backupBatch, and an error, if there is any.
func (c *FakeBackupBatches) Update(backupBatch *v1beta1.BackupBatch) (result *v1beta1.BackupBatch, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(backupbatchesResource, c.ns, backupBatch), &v1beta1.BackupBatch{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.BackupBatch), err
}

// Delete takes name of the backupBatch and deletes it. Returns an error if one occurs.
func (c *FakeBackupBatches) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(backupbatchesResource, c.ns, name), &v1beta1.BackupBatch{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeBackupBatches) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(backupbatchesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1beta1.BackupBatchList{})
	return err
}

// Patch applies the patch and returns the patched backupBatch.
func (c *FakeBackupBatches) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1beta1.BackupBatch, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(backupbatchesResource, c.ns, name, pt, data, subresources...), &v1beta1.BackupBatch{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1beta1.BackupBatch), err
}
