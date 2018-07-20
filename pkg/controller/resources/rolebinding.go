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

package resources

import (
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
)

// RoleBinding Embedding and extending RoleBinding interface
type RoleBinding struct {
	*rbacv1.RoleBinding
}

// Create interface
func (s *RoleBinding) Create(c *k8s.Clientset) (ResourceHandle, error) {
	obj, err := c.RbacV1().RoleBindings(s.GetNamespace()).Create(s.RoleBinding)
	return &RoleBinding{RoleBinding: obj}, err
}

// Delete interface
func (s *RoleBinding) Delete(c *k8s.Clientset) error {
	deletePropagationBackgroundPolicy := metav1.DeletePropagationBackground
	options := &metav1.DeleteOptions{
		PropagationPolicy: &deletePropagationBackgroundPolicy,
	}
	return c.RbacV1().RoleBindings(s.GetNamespace()).Delete(s.GetName(), options)
}

// Update interface
func (s *RoleBinding) Update(c *k8s.Clientset) error {
	_, err := c.RbacV1().RoleBindings(s.GetNamespace()).Update(s.RoleBinding)
	return err
}

// Get interface
func (s *RoleBinding) Get(c *k8s.Clientset) (ResourceHandle, error) {
	options := metav1.GetOptions{ResourceVersion: ""}
	obj, err := c.RbacV1().RoleBindings(s.GetNamespace()).Get(s.GetName(), options)
	return &RoleBinding{RoleBinding: obj}, err
}
