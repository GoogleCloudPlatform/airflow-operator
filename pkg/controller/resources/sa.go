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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
)

// ServiceAccount Embedding and extending ServiceAccount interface
type ServiceAccount struct {
	*corev1.ServiceAccount
}

// Create interface
func (s *ServiceAccount) Create(c *k8s.Clientset) (ResourceHandle, error) {
	obj, err := c.CoreV1().ServiceAccounts(s.GetNamespace()).Create(s.ServiceAccount)
	return &ServiceAccount{ServiceAccount: obj}, err
}

// Delete interface
func (s *ServiceAccount) Delete(c *k8s.Clientset) error {
	deletePropagationBackgroundPolicy := metav1.DeletePropagationBackground
	options := &metav1.DeleteOptions{
		PropagationPolicy: &deletePropagationBackgroundPolicy,
	}
	return c.CoreV1().ServiceAccounts(s.GetNamespace()).Delete(s.GetName(), options)
}

// Update interface
func (s *ServiceAccount) Update(c *k8s.Clientset) error {
	_, err := c.CoreV1().ServiceAccounts(s.GetNamespace()).Update(s.ServiceAccount)
	return err
}

// Get interface
func (s *ServiceAccount) Get(c *k8s.Clientset) (ResourceHandle, error) {
	options := metav1.GetOptions{ResourceVersion: ""}
	obj, err := c.CoreV1().ServiceAccounts(s.GetNamespace()).Get(s.GetName(), options)
	return &ServiceAccount{ServiceAccount: obj}, err
}
