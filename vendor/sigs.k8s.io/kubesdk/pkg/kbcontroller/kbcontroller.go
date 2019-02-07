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

package kbcontroller

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	cr "sigs.k8s.io/kubesdk/pkg/customresource"
)

// CreateController creates a new Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller and Start it when the Manager is Started.
func CreateController(name string, mgr manager.Manager, handle cr.Handle, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(name+"-ctrl", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Base resource
	err = c.Watch(&source.Kind{Type: handle.(runtime.Object)},
		&handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	/*
		// TODO(user): Modify this to be the types you create
		// Uncomment watch a Deployment created by ESCluster - change this for objects you create
		err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
			IsController: true,
			OwnerType:    &elasticsearchv1alpha1.ESCluster{},
		})
		if err != nil {
			return err
		}
	*/

	return nil
}
