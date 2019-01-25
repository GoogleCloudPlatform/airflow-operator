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

package application

import (
	app "github.com/kubernetes-sigs/application/pkg/apis/app/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/kubesdk/pkg/component"
	"sigs.k8s.io/kubesdk/pkg/resource"
	"sigs.k8s.io/kubesdk/pkg/resource/manager/k8s"
)

// Application obj to attach methods
type Application struct {
	app.Application
}

// SetSelector attaches selectors to Application object
func (a *Application) SetSelector(labels map[string]string) *Application {
	a.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	return a
}

// SetName sets name
func (a *Application) SetName(value string) *Application {
	a.ObjectMeta.Name = value
	return a
}

// SetNamespace asets namespace
func (a *Application) SetNamespace(value string) *Application {
	a.ObjectMeta.Namespace = value
	return a
}

// AddLabels adds more labels
func (a *Application) AddLabels(value component.KVMap) *Application {
	value.Merge(a.ObjectMeta.Labels)
	a.ObjectMeta.Labels = value
	return a
}

// Observable returns resource object
func (a *Application) Observable() *resource.Observable {
	return &resource.Observable{
		Obj: k8s.Observable{
			Obj:     &a.Application,
			ObjList: &app.ApplicationList{},
			Labels:  a.GetLabels(),
		},
		Type: k8s.Type,
	}
}

// Item returns resource object
func (a *Application) Item() *resource.Item {
	return &resource.Item{
		Lifecycle: resource.LifecycleManaged,
		Type:      k8s.Type,
		Obj: &k8s.Object{
			Obj:     &a.Application,
			ObjList: &app.ApplicationList{},
		},
	}
}

// AddToScheme return AddToScheme of application crd
func AddToScheme(sb *runtime.SchemeBuilder) {
	*sb = append(*sb, app.AddToScheme)
}

// SetComponentGK attaches component GK to Application object
func (a *Application) SetComponentGK(bag *resource.Bag) *Application {
	a.Spec.ComponentGroupKinds = []metav1.GroupKind{}
	gkmap := map[schema.GroupKind]struct{}{}
	for _, item := range bag.ByType(k8s.Type) {
		obj := item.Obj.(*k8s.Object)
		if obj.ObjList != nil {
			ro := obj.Obj.(runtime.Object)
			gk := ro.GetObjectKind().GroupVersionKind().GroupKind()
			if _, ok := gkmap[gk]; !ok {
				gkmap[gk] = struct{}{}
				mgk := metav1.GroupKind{
					Group: gk.Group,
					Kind:  gk.Kind,
				}
				a.Spec.ComponentGroupKinds = append(a.Spec.ComponentGroupKinds, mgk)
			}
		}
	}
	return a
}
