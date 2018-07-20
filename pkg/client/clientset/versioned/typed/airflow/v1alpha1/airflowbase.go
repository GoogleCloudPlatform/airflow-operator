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

// AirflowBasesGetter has a method to return a AirflowBaseInterface.
// A group's client should implement this interface.
type AirflowBasesGetter interface {
	AirflowBases(namespace string) AirflowBaseInterface
}

// AirflowBaseInterface has methods to work with AirflowBase resources.
type AirflowBaseInterface interface {
	Create(*v1alpha1.AirflowBase) (*v1alpha1.AirflowBase, error)
	Update(*v1alpha1.AirflowBase) (*v1alpha1.AirflowBase, error)
	UpdateStatus(*v1alpha1.AirflowBase) (*v1alpha1.AirflowBase, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.AirflowBase, error)
	List(opts v1.ListOptions) (*v1alpha1.AirflowBaseList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.AirflowBase, err error)
	AirflowBaseExpansion
}

// airflowBases implements AirflowBaseInterface
type airflowBases struct {
	client rest.Interface
	ns     string
}

// newAirflowBases returns a AirflowBases
func newAirflowBases(c *AirflowV1alpha1Client, namespace string) *airflowBases {
	return &airflowBases{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the airflowBase, and returns the corresponding airflowBase object, and an error if there is any.
func (c *airflowBases) Get(name string, options v1.GetOptions) (result *v1alpha1.AirflowBase, err error) {
	result = &v1alpha1.AirflowBase{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("airflowbases").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of AirflowBases that match those selectors.
func (c *airflowBases) List(opts v1.ListOptions) (result *v1alpha1.AirflowBaseList, err error) {
	result = &v1alpha1.AirflowBaseList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("airflowbases").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested airflowBases.
func (c *airflowBases) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("airflowbases").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a airflowBase and creates it.  Returns the server's representation of the airflowBase, and an error, if there is any.
func (c *airflowBases) Create(airflowBase *v1alpha1.AirflowBase) (result *v1alpha1.AirflowBase, err error) {
	result = &v1alpha1.AirflowBase{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("airflowbases").
		Body(airflowBase).
		Do().
		Into(result)
	return
}

// Update takes the representation of a airflowBase and updates it. Returns the server's representation of the airflowBase, and an error, if there is any.
func (c *airflowBases) Update(airflowBase *v1alpha1.AirflowBase) (result *v1alpha1.AirflowBase, err error) {
	result = &v1alpha1.AirflowBase{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("airflowbases").
		Name(airflowBase.Name).
		Body(airflowBase).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *airflowBases) UpdateStatus(airflowBase *v1alpha1.AirflowBase) (result *v1alpha1.AirflowBase, err error) {
	result = &v1alpha1.AirflowBase{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("airflowbases").
		Name(airflowBase.Name).
		SubResource("status").
		Body(airflowBase).
		Do().
		Into(result)
	return
}

// Delete takes name of the airflowBase and deletes it. Returns an error if one occurs.
func (c *airflowBases) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("airflowbases").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *airflowBases) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("airflowbases").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched airflowBase.
func (c *airflowBases) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.AirflowBase, err error) {
	result = &v1alpha1.AirflowBase{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("airflowbases").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
