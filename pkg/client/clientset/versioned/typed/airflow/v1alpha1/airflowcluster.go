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
package v1alpha1

import (
	v1alpha1 "k8s.io/airflow-operator/pkg/apis/airflow/v1alpha1"
	scheme "k8s.io/airflow-operator/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// AirflowClustersGetter has a method to return a AirflowClusterInterface.
// A group's client should implement this interface.
type AirflowClustersGetter interface {
	AirflowClusters(namespace string) AirflowClusterInterface
}

// AirflowClusterInterface has methods to work with AirflowCluster resources.
type AirflowClusterInterface interface {
	Create(*v1alpha1.AirflowCluster) (*v1alpha1.AirflowCluster, error)
	Update(*v1alpha1.AirflowCluster) (*v1alpha1.AirflowCluster, error)
	UpdateStatus(*v1alpha1.AirflowCluster) (*v1alpha1.AirflowCluster, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.AirflowCluster, error)
	List(opts v1.ListOptions) (*v1alpha1.AirflowClusterList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.AirflowCluster, err error)
	AirflowClusterExpansion
}

// airflowClusters implements AirflowClusterInterface
type airflowClusters struct {
	client rest.Interface
	ns     string
}

// newAirflowClusters returns a AirflowClusters
func newAirflowClusters(c *AirflowV1alpha1Client, namespace string) *airflowClusters {
	return &airflowClusters{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the airflowCluster, and returns the corresponding airflowCluster object, and an error if there is any.
func (c *airflowClusters) Get(name string, options v1.GetOptions) (result *v1alpha1.AirflowCluster, err error) {
	result = &v1alpha1.AirflowCluster{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("airflowclusters").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of AirflowClusters that match those selectors.
func (c *airflowClusters) List(opts v1.ListOptions) (result *v1alpha1.AirflowClusterList, err error) {
	result = &v1alpha1.AirflowClusterList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("airflowclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested airflowClusters.
func (c *airflowClusters) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("airflowclusters").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a airflowCluster and creates it.  Returns the server's representation of the airflowCluster, and an error, if there is any.
func (c *airflowClusters) Create(airflowCluster *v1alpha1.AirflowCluster) (result *v1alpha1.AirflowCluster, err error) {
	result = &v1alpha1.AirflowCluster{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("airflowclusters").
		Body(airflowCluster).
		Do().
		Into(result)
	return
}

// Update takes the representation of a airflowCluster and updates it. Returns the server's representation of the airflowCluster, and an error, if there is any.
func (c *airflowClusters) Update(airflowCluster *v1alpha1.AirflowCluster) (result *v1alpha1.AirflowCluster, err error) {
	result = &v1alpha1.AirflowCluster{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("airflowclusters").
		Name(airflowCluster.Name).
		Body(airflowCluster).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *airflowClusters) UpdateStatus(airflowCluster *v1alpha1.AirflowCluster) (result *v1alpha1.AirflowCluster, err error) {
	result = &v1alpha1.AirflowCluster{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("airflowclusters").
		Name(airflowCluster.Name).
		SubResource("status").
		Body(airflowCluster).
		Do().
		Into(result)
	return
}

// Delete takes name of the airflowCluster and deletes it. Returns an error if one occurs.
func (c *airflowClusters) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("airflowclusters").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *airflowClusters) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("airflowclusters").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched airflowCluster.
func (c *airflowClusters) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.AirflowCluster, err error) {
	result = &v1alpha1.AirflowCluster{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("airflowclusters").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
