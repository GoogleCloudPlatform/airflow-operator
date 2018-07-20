/*
Copyright 2018 Google LLC
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
	v1alpha1 "k8s.io/airflow-operator/pkg/apis/airflow/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeAirflowBases implements AirflowBaseInterface
type FakeAirflowBases struct {
	Fake *FakeAirflowV1alpha1
	ns   string
}

var airflowbasesResource = schema.GroupVersionResource{Group: "airflow.k8s.io", Version: "v1alpha1", Resource: "airflowbases"}

var airflowbasesKind = schema.GroupVersionKind{Group: "airflow.k8s.io", Version: "v1alpha1", Kind: "AirflowBase"}

// Get takes name of the airflowBase, and returns the corresponding airflowBase object, and an error if there is any.
func (c *FakeAirflowBases) Get(name string, options v1.GetOptions) (result *v1alpha1.AirflowBase, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(airflowbasesResource, c.ns, name), &v1alpha1.AirflowBase{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AirflowBase), err
}

// List takes label and field selectors, and returns the list of AirflowBases that match those selectors.
func (c *FakeAirflowBases) List(opts v1.ListOptions) (result *v1alpha1.AirflowBaseList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(airflowbasesResource, airflowbasesKind, c.ns, opts), &v1alpha1.AirflowBaseList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.AirflowBaseList{}
	for _, item := range obj.(*v1alpha1.AirflowBaseList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested airflowBases.
func (c *FakeAirflowBases) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(airflowbasesResource, c.ns, opts))

}

// Create takes the representation of a airflowBase and creates it.  Returns the server's representation of the airflowBase, and an error, if there is any.
func (c *FakeAirflowBases) Create(airflowBase *v1alpha1.AirflowBase) (result *v1alpha1.AirflowBase, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(airflowbasesResource, c.ns, airflowBase), &v1alpha1.AirflowBase{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AirflowBase), err
}

// Update takes the representation of a airflowBase and updates it. Returns the server's representation of the airflowBase, and an error, if there is any.
func (c *FakeAirflowBases) Update(airflowBase *v1alpha1.AirflowBase) (result *v1alpha1.AirflowBase, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(airflowbasesResource, c.ns, airflowBase), &v1alpha1.AirflowBase{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AirflowBase), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAirflowBases) UpdateStatus(airflowBase *v1alpha1.AirflowBase) (*v1alpha1.AirflowBase, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(airflowbasesResource, "status", c.ns, airflowBase), &v1alpha1.AirflowBase{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AirflowBase), err
}

// Delete takes name of the airflowBase and deletes it. Returns an error if one occurs.
func (c *FakeAirflowBases) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(airflowbasesResource, c.ns, name), &v1alpha1.AirflowBase{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAirflowBases) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(airflowbasesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.AirflowBaseList{})
	return err
}

// Patch applies the patch and returns the patched airflowBase.
func (c *FakeAirflowBases) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.AirflowBase, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(airflowbasesResource, c.ns, name, data, subresources...), &v1alpha1.AirflowBase{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AirflowBase), err
}
