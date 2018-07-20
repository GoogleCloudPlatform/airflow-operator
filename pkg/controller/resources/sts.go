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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
)

// StatefulSet Embedding and extending statefulset interface
type StatefulSet struct {
	*appsv1.StatefulSet
}

// Create interface
func (s *StatefulSet) Create(c *k8s.Clientset) (ResourceHandle, error) {
	obj, err := c.AppsV1().StatefulSets(s.GetNamespace()).Create(s.StatefulSet)
	return &StatefulSet{StatefulSet: obj}, err
}

// Delete interface
func (s *StatefulSet) Delete(c *k8s.Clientset) error {
	deletePropagationBackgroundPolicy := metav1.DeletePropagationBackground
	options := &metav1.DeleteOptions{
		PropagationPolicy: &deletePropagationBackgroundPolicy,
	}
	return c.AppsV1().StatefulSets(s.GetNamespace()).Delete(s.GetName(), options)
}

// Update interface
func (s *StatefulSet) Update(c *k8s.Clientset) error {
	_, err := c.AppsV1().StatefulSets(s.GetNamespace()).Update(s.StatefulSet)
	return err
}

// Get interface
func (s *StatefulSet) Get(c *k8s.Clientset) (ResourceHandle, error) {
	options := metav1.GetOptions{ResourceVersion: ""}
	obj, err := c.AppsV1().StatefulSets(s.GetNamespace()).Get(s.GetName(), options)
	return &StatefulSet{StatefulSet: obj}, err
}
