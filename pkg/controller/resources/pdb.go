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
	policyv1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
)

// PodDisruptionBudget Embedding and extending PodDisruptionBudget interface
type PodDisruptionBudget struct {
	*policyv1.PodDisruptionBudget
}

// Create interface
func (s *PodDisruptionBudget) Create(c *k8s.Clientset) (ResourceHandle, error) {
	obj, err := c.PolicyV1beta1().PodDisruptionBudgets(s.GetNamespace()).Create(s.PodDisruptionBudget)
	return &PodDisruptionBudget{PodDisruptionBudget: obj}, err
}

// Delete interface
func (s *PodDisruptionBudget) Delete(c *k8s.Clientset) error {
	deletePropagationBackgroundPolicy := metav1.DeletePropagationBackground
	options := &metav1.DeleteOptions{
		PropagationPolicy: &deletePropagationBackgroundPolicy,
	}
	return c.PolicyV1beta1().PodDisruptionBudgets(s.GetNamespace()).Delete(s.GetName(), options)
}

// Update interface
func (s *PodDisruptionBudget) Update(c *k8s.Clientset) error {
	_, err := c.PolicyV1beta1().PodDisruptionBudgets(s.GetNamespace()).Update(s.PodDisruptionBudget)
	return err
}

// Get interface
func (s *PodDisruptionBudget) Get(c *k8s.Clientset) (ResourceHandle, error) {
	options := metav1.GetOptions{ResourceVersion: ""}
	obj, err := c.PolicyV1beta1().PodDisruptionBudgets(s.GetNamespace()).Get(s.GetName(), options)
	return &PodDisruptionBudget{PodDisruptionBudget: obj}, err
}
