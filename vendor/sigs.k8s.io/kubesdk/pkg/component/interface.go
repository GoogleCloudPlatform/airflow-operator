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

package component

import (
	"sigs.k8s.io/kubesdk/pkg/resource"
	"sigs.k8s.io/kubesdk/pkg/resource/manager"
	"time"
)

// Handle is an interface for operating on logical Components of a CR
type Handle interface {
	ExpectedResources(rsrc interface{}, labels map[string]string, dependent, aggregated *resource.Bag) (*resource.Bag, error)
}

// StatusInterface - interface to update compoennt status
type StatusInterface interface {
	UpdateComponentStatus(rsrc interface{}, reconciled *resource.Bag, err error) time.Duration
}

// FinalizeInterface - finalize component
type FinalizeInterface interface {
	Finalize(rsrc interface{}, dependent, observed *resource.Bag) error
}

// DiffersInterface - call differs
type DiffersInterface interface {
	Differs(expected resource.Item, observed resource.Item) bool
}

// DependentResourcesInterface - get dependent resources
type DependentResourcesInterface interface {
	DependentResources(rsrc interface{}) *resource.Bag
}

// MutateInterface - interface to mutate objects
type MutateInterface interface {
	Mutate(rsrc interface{}, rsrclabels map[string]string, expected, dependent, observed *resource.Bag) (*resource.Bag, error)
}

// ObservablesInterface - interface to get observables
type ObservablesInterface interface {
	Observables(rsrcmgr *manager.ResourceManager, rsrc interface{}, labels map[string]string, expected *resource.Bag) []resource.Observable
}
