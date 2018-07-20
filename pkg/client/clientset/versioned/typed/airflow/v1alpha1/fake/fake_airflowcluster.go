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

// FakeAirflowClusters implements AirflowClusterInterface
type FakeAirflowClusters struct {
	Fake *FakeAirflowV1alpha1
	ns   string
}

var airflowclustersResource = schema.GroupVersionResource{Group: "airflow.k8s.io", Version: "v1alpha1", Resource: "airflowclusters"}

var airflowclustersKind = schema.GroupVersionKind{Group: "airflow.k8s.io", Version: "v1alpha1", Kind: "AirflowCluster"}

// Get takes name of the airflowCluster, and returns the corresponding airflowCluster object, and an error if there is any.
func (c *FakeAirflowClusters) Get(name string, options v1.GetOptions) (result *v1alpha1.AirflowCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(airflowclustersResource, c.ns, name), &v1alpha1.AirflowCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AirflowCluster), err
}

// List takes label and field selectors, and returns the list of AirflowClusters that match those selectors.
func (c *FakeAirflowClusters) List(opts v1.ListOptions) (result *v1alpha1.AirflowClusterList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(airflowclustersResource, airflowclustersKind, c.ns, opts), &v1alpha1.AirflowClusterList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.AirflowClusterList{}
	for _, item := range obj.(*v1alpha1.AirflowClusterList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested airflowClusters.
func (c *FakeAirflowClusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(airflowclustersResource, c.ns, opts))

}

// Create takes the representation of a airflowCluster and creates it.  Returns the server's representation of the airflowCluster, and an error, if there is any.
func (c *FakeAirflowClusters) Create(airflowCluster *v1alpha1.AirflowCluster) (result *v1alpha1.AirflowCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(airflowclustersResource, c.ns, airflowCluster), &v1alpha1.AirflowCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AirflowCluster), err
}

// Update takes the representation of a airflowCluster and updates it. Returns the server's representation of the airflowCluster, and an error, if there is any.
func (c *FakeAirflowClusters) Update(airflowCluster *v1alpha1.AirflowCluster) (result *v1alpha1.AirflowCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(airflowclustersResource, c.ns, airflowCluster), &v1alpha1.AirflowCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AirflowCluster), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeAirflowClusters) UpdateStatus(airflowCluster *v1alpha1.AirflowCluster) (*v1alpha1.AirflowCluster, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(airflowclustersResource, "status", c.ns, airflowCluster), &v1alpha1.AirflowCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AirflowCluster), err
}

// Delete takes name of the airflowCluster and deletes it. Returns an error if one occurs.
func (c *FakeAirflowClusters) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(airflowclustersResource, c.ns, name), &v1alpha1.AirflowCluster{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeAirflowClusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(airflowclustersResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.AirflowClusterList{})
	return err
}

// Patch applies the patch and returns the patched airflowCluster.
func (c *FakeAirflowClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.AirflowCluster, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(airflowclustersResource, c.ns, name, data, subresources...), &v1alpha1.AirflowCluster{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.AirflowCluster), err
}
