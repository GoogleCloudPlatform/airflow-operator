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

package manager

import (
	"sigs.k8s.io/controller-reconciler/pkg/object"
)

// Manager is an interface for operating on CRs
type Manager interface {
	Observe(observables ...object.Observable) (*object.Bag, error)
	Update(item object.Item) error
	Create(item object.Item) error
	Delete(item object.Item) error
	SpecDiffers(expected, observed *object.Item) bool
	ObservablesFromObjects(bag *object.Bag, labels map[string]string) []object.Observable
}
