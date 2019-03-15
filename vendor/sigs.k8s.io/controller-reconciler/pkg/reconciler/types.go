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

package reconciler

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Common const definitions
const (
	LifecycleManaged  = "managed"
	LifecycleReferred = "referred"
	LifecycleNoUpdate = "noupdate"
)

// Interface -
type Interface interface {
	GetName() string
	IsSameAs(interface{}) bool
	SetOwnerReferences(*metav1.OwnerReference) bool
}

// Object is a container to capture the k8s resource info to be used by controller
type Object struct {
	// Lifecycle can be: managed, reference
	Lifecycle string
	// Type - object type
	Type string
	// Obj -  object
	Obj Interface
	// Delete - marker for deletion
	Delete bool
}

// Observable captures the k8s resource info and selector to fetch child resources
type Observable struct {
	// Type - object type
	Type string
	// Obj - object
	Obj interface{}
}

// KVMap is a map[string]string
type KVMap map[string]string
