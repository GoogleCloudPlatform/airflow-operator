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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"sigs.k8s.io/kubesdk/pkg/finalizer"
	"sigs.k8s.io/kubesdk/pkg/resource"
	"sigs.k8s.io/kubesdk/pkg/resource/manager"
	"strings"
	"time"
)

// Constants defining labels
const (
	LabelCR          = "custom-resource"
	LabelCRName      = "custom-resource-name"
	LabelCRNamespace = "custom-resource-namespace"
	LabelComponent   = "component"
)

// Labels return
func Labels(cr metav1.Object, component string) map[string]string {
	return map[string]string{
		LabelCR:          strings.Trim(reflect.TypeOf(cr).String(), "*"),
		LabelCRName:      cr.GetName(),
		LabelCRNamespace: cr.GetNamespace(),
		LabelComponent:   component,
	}
}

// Labels return the common labels for a resource
func (c *Component) Labels() map[string]string {
	return Labels(c.CR, c.Name)
}

// Merge is used to merge multiple maps into the target map
func (out KVMap) Merge(kvmaps ...KVMap) {
	for _, kvmap := range kvmaps {
		for k, v := range kvmap {
			out[k] = v
		}
	}
}

// DependentResources Get dependent resources from component or defaults
func (c *Component) DependentResources() *resource.Bag {
	if s, ok := c.Handle.(DependentResourcesInterface); ok {
		return s.DependentResources(c.CR)
	}
	return &resource.Bag{}
}

// Mutate Get dependent resources from component or defaults
func (c *Component) Mutate(expected, dependent, observed *resource.Bag) (*resource.Bag, error) {
	if s, ok := c.Handle.(MutateInterface); ok {
		return s.Mutate(c.CR, c.Labels(), expected, dependent, observed)
	}
	return expected, nil
}

// GetAllObservables - get all observables
func (c *Component) GetAllObservables(rsrcmgr *manager.ResourceManager, bag *resource.Bag) []resource.Observable {
	var observables []resource.Observable
	for _, m := range rsrcmgr.All() {
		o := m.ObservablesFromObjects(bag, c.Labels())
		observables = append(observables, o...)
	}
	return observables
}

// Observables - gets observbables from the expected objects
func (c *Component) Observables(rsrcmgr *manager.ResourceManager, expected *resource.Bag) []resource.Observable {
	if s, ok := c.Handle.(ObservablesInterface); ok {
		return s.Observables(rsrcmgr, c.CR, c.Labels(), expected)
	}
	return c.GetAllObservables(rsrcmgr, expected)
}

// Differs - call differs
func (c *Component) Differs(expected resource.Item, observed resource.Item) bool {
	if s, ok := c.Handle.(DiffersInterface); ok {
		return s.Differs(expected, observed)
	}
	return true
}

// Finalize - Finalize component
func (c *Component) Finalize(observed, dependent *resource.Bag) error {
	if s, ok := c.Handle.(FinalizeInterface); ok {
		return s.Finalize(c.CR, observed, dependent)
	}
	r := c.CR.(metav1.Object)
	finalizer.RemoveStandard(r)
	return nil
}

// UpdateComponentStatus - update component status
func (c *Component) UpdateComponentStatus(reconciled *resource.Bag, err error) time.Duration {
	var period time.Duration
	if s, ok := c.Handle.(StatusInterface); ok {
		return s.UpdateComponentStatus(c.CR, reconciled, err)
	}
	return period
}
