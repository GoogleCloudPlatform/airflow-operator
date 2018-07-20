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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8s "k8s.io/client-go/kubernetes"
)

// ResourceHandle exposes CRUD for k8s resource objects
type ResourceHandle interface {
	Create(c *k8s.Clientset) (ResourceHandle, error)
	Delete(c *k8s.Clientset) error
	Update(c *k8s.Clientset) error
	Get(c *k8s.Clientset) (ResourceHandle, error)
	SetResourceVersion(version string)
	GetResourceVersion() string
	GetNamespace() string
	GetName() string
	GetSelfLink() string
	GetOwnerReferences() []metav1.OwnerReference
}
