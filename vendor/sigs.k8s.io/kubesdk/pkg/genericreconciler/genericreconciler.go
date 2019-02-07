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

package genericreconciler

import (
	"fmt"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	urt "k8s.io/apimachinery/pkg/util/runtime"
	"log"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	app "sigs.k8s.io/kubesdk/pkg/application"
	"sigs.k8s.io/kubesdk/pkg/component"
	cr "sigs.k8s.io/kubesdk/pkg/customresource"
	"sigs.k8s.io/kubesdk/pkg/resource"
	"sigs.k8s.io/kubesdk/pkg/resource/manager"
	"sigs.k8s.io/kubesdk/pkg/resource/manager/k8s"
	"time"
)

// Constants
const (
	DefaultReconcilePeriod      = 3 * time.Minute
	FinalizeReconcilePeriod     = 30 * time.Second
	CRGetFailureReconcilePeriod = 30 * time.Second
)

func handleErrorArr(info string, name string, e error, errs []error) []error {
	HandleError(info, name, e)
	return append(errs, e)
}

// HandleError common error handling routine
func HandleError(info string, name string, e error) error {
	urt.HandleError(fmt.Errorf("Failed: [%s] %s. %s", name, info, e.Error()))
	return e
}

func (gr *Reconciler) itemMgr(i resource.Item) (manager.Manager, error) {
	m := gr.RsrcMgr.Get(i.Type)
	if m == nil {
		return m, fmt.Errorf("Resource Manager not registered for Type: %s", i.Type)
	}
	return m, nil
}

// ReconcileCR is a generic function that reconciles expected and observed resources
func (gr *Reconciler) ReconcileCR(namespacedname types.NamespacedName) (reconcile.Result, error) {
	var p time.Duration
	period := DefaultReconcilePeriod
	expected := &resource.Bag{}
	rsrc := gr.CR.Handle.(runtime.Object).DeepCopyObject().(cr.Handle)
	name := reflect.TypeOf(rsrc).String() + "/" + namespacedname.String()
	rm := gr.RsrcMgr.Get("k8s")
	err := k8s.Get(rm, namespacedname, rsrc.(runtime.Object))
	crhandle := cr.CustomResource{Handle: rsrc}
	if err != nil {
		if apierror.IsNotFound(err) {
			urt.HandleError(fmt.Errorf("not found %s. %s", name, err.Error()))
			return reconcile.Result{}, nil
		}
		return reconcile.Result{RequeueAfter: CRGetFailureReconcilePeriod}, err
	}
	o := rsrc.(metav1.Object)
	log.Printf("%s Validating spec\n", name)
	err = crhandle.Validate()
	if err == nil {
		log.Printf("%s Applying defaults\n", name)
		crhandle.ApplyDefaults()
		components := rsrc.Components()
		for _, component := range components {
			if o.GetDeletionTimestamp() == nil {
				p, err = gr.ReconcileComponent(name, component, expected)
			} else {
				err = gr.FinalizeComponent(name, component, expected)
				p = FinalizeReconcilePeriod
			}
			if p != 0 && p < period {
				period = p
			}
		}
	}

	if err != nil {
		urt.HandleError(fmt.Errorf("error reconciling %s. %s", name, err.Error()))
		rsrc.HandleError(err)
	}
	err = rm.Update(resource.Item{Obj: &k8s.Object{Obj: rsrc.(metav1.Object)}})
	if err != nil {
		urt.HandleError(fmt.Errorf("error updating %s. %s", name, err.Error()))
	}
	return reconcile.Result{RequeueAfter: period}, err
}

func (gr *Reconciler) observe(observables []resource.Observable) (*resource.Bag, error) {
	seen := &resource.Bag{}
	for _, m := range gr.RsrcMgr.All() {
		o, err := m.Observe(observables...)
		if err != nil {
			return &resource.Bag{}, err
		}
		seen.Merge(o)
	}
	return seen, nil
}

// ObserveAndMutate is a function that is called to observe and mutate expected resources
func (gr *Reconciler) ObserveAndMutate(crname string, c component.Component, mutate bool, aggregated *resource.Bag) (*resource.Bag, *resource.Bag, *resource.Bag, string, error) {
	var err error
	var expected, observed, dependent *resource.Bag

	// Get dependenta objects

	// Loop over all RMs
	stage := "dependent resources"
	dependent, err = gr.observe(c.GetAllObservables(&gr.RsrcMgr, c.DependentResources()))
	if err == nil && dependent != nil {
		// Get Expected resources
		stage = "gathering expected resources"
		expected, err = c.ExpectedResources(c.CR, c.Labels(), dependent, aggregated)
		if err == nil && expected != nil {
			// Get Observe observables
			stage = "observing resources"
			observed, err = gr.observe(c.Observables(&gr.RsrcMgr, expected))
			if mutate && err == nil {
				// Mutate expected objects
				stage = "mutating resources"
				expected, err = c.Mutate(expected, dependent, observed)
				if err == nil && expected != nil {
					// Get observables
					// Observe observables
					stage = "observing resources after mutation"
					observed, err = gr.observe(c.Observables(&gr.RsrcMgr, expected))
				}
			}
		}
	}
	if err != nil {
		observed = &resource.Bag{}
		expected = &resource.Bag{}
	}
	if expected == nil {
		expected = &resource.Bag{}
	}
	if observed == nil {
		observed = &resource.Bag{}
	}
	return expected, observed, dependent, stage, err
}

// FinalizeComponent is a function that finalizes component
func (gr *Reconciler) FinalizeComponent(crname string, c component.Component, aggregated *resource.Bag) error {
	cname := crname + "(cmpnt:" + c.Name + ")"
	log.Printf("%s  { finalizing component\n", cname)
	defer log.Printf("%s  } finalizing component\n", cname)

	expected, observed, dependent, stage, err := gr.ObserveAndMutate(crname, c, false, aggregated)

	if err != nil {
		HandleError(stage, crname, err)
	}
	aggregated.Add(expected.Items()...)
	stage = "finalizing resource"
	err = c.Finalize(observed, dependent)
	if err == nil {
		stage = "finalizing deletion of objects"
		for _, o := range observed.Items() {
			oRsrcName := o.Obj.GetName()
			if o.Delete {
				if rm, e := gr.itemMgr(o); e != nil {
					err = e
					break
				} else if e := rm.Delete(o); e != nil {
					err = e
					break
				} else {
					log.Printf("%s   -delete: %s\n", cname, oRsrcName)
				}

			}
		}
	}
	if err != nil {
		HandleError(stage, crname, err)
	}
	return err
}

// ReconcileComponent is a generic function that reconciles expected and observed resources
func (gr *Reconciler) ReconcileComponent(crname string, c component.Component, aggregated *resource.Bag) (time.Duration, error) {
	errs := []error{}
	var reconciled *resource.Bag = new(resource.Bag)

	cname := crname + "(cmpnt:" + c.Name + ")"
	log.Printf("%s  { reconciling component\n", cname)
	defer log.Printf("%s  } reconciling component\n", cname)

	expected, observed, _, stage, err := gr.ObserveAndMutate(crname, c, true, aggregated)

	// Reconciliation logic is straight-forward:
	// This method gets the list of expected resources and observed resources
	// We compare the 2 lists and:
	//  create(rsrc) where rsrc is in expected but not in observed
	//  delete(rsrc) where rsrc is in observed but not in expected
	//  update(rsrc) where rsrc is in observed and expected
	//
	// We have a notion of Managed and Referred resources
	// Only Managed resources are CRUD'd
	// Missing Reffered resources are treated as errors and surfaced as such in the status field
	//

	if err != nil {
		errs = handleErrorArr(stage, crname, err, errs)
	} else {
		aggregated.Add(expected.Items()...)
		log.Printf("%s  Expected Resources:\n", cname)
		for _, e := range expected.Items() {
			log.Printf("%s   exp: %s %s\n", cname, e.Type, e.Obj.GetName())
		}
		log.Printf("%s  Observed Resources:\n", cname)
		for _, e := range observed.Items() {
			log.Printf("%s   obs: %s\n", cname, e.Obj.GetName())
		}

		log.Printf("%s  Reconciling Resources:\n", cname)
	}
	for _, e := range expected.Items() {
		seen := false
		eRsrcName := e.Obj.GetName()
		for _, o := range observed.Items() {
			if e.Type != o.Type || !e.Obj.IsSameAs(o.Obj) {
				continue
			}
			// rsrc is seen in both expected and observed, update it if needed
			reconciled.Add(o)
			seen = true

			if e.Lifecycle != resource.LifecycleManaged {
				log.Printf("%s   notmanaged: %s\n", cname, eRsrcName)
				break
			}

			rm, err := gr.itemMgr(e)
			if err != nil {
				errs = handleErrorArr("update", eRsrcName, err, errs)
				break
			}

			// Component Differs is not expected to mutate e based on o
			compDiffers := c.Differs(e, o)
			// Resource Manager Differs can mutate e based on o
			rmDiffers := rm.SpecDiffers(&e, &o)
			refchange := e.Obj.SetOwnerReferences(c.OwnerRef)
			if rmDiffers && compDiffers || refchange {
				if err := rm.Update(e); err != nil {
					errs = handleErrorArr("update", eRsrcName, err, errs)
				} else {
					log.Printf("%s   update: %s\n", cname, eRsrcName)
				}
			} else {
				log.Printf("%s   nochange: %s\n", cname, eRsrcName)
			}
			break
		}
		// rsrc is in expected but not in observed - create
		if !seen {
			if e.Lifecycle == resource.LifecycleManaged {
				e.Obj.SetOwnerReferences(c.OwnerRef)
				if rm, err := gr.itemMgr(e); err != nil {
					errs = handleErrorArr("Create", cname, err, errs)
				} else if err := rm.Create(e); err != nil {
					errs = handleErrorArr("Create", cname, err, errs)
				} else {
					log.Printf("%s   +create: %s\n", cname, eRsrcName)
					reconciled.Add(e)
				}
			} else {
				err := fmt.Errorf("missing resource not managed by %s: %s", cname, eRsrcName)
				errs = handleErrorArr("missing resource", cname, err, errs)
			}
		}
	}

	// delete(observed - expected)
	for _, o := range observed.Items() {
		seen := false
		oRsrcName := o.Obj.GetName()
		for _, e := range expected.Items() {
			if e.Type == o.Type && e.Obj.IsSameAs(o.Obj) {
				seen = true
				break
			}
		}
		// rsrc is in observed but not in expected - delete
		if !seen {
			if rm, err := gr.itemMgr(o); err != nil {
				errs = handleErrorArr("delete", oRsrcName, err, errs)
			} else if err := rm.Delete(o); err != nil {
				errs = handleErrorArr("delete", oRsrcName, err, errs)
			} else {
				log.Printf("%s   -delete: %s\n", cname, oRsrcName)
			}
		}
	}

	err = utilerrors.NewAggregate(errs)
	period := c.UpdateComponentStatus(reconciled, err)
	return period, err
}

// Reconcile expected by kubebuilder
func (gr *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	r, err := gr.ReconcileCR(request.NamespacedName)
	if err != nil {
		fmt.Printf("err: %s", err.Error())
	}
	return r, err
}

// AddToSchemes for adding Application to scheme
var AddToSchemes runtime.SchemeBuilder

// Init sets up Reconciler
func (gr *Reconciler) Init() {
	km := k8s.NewRsrcManager().WithName("basek8s").WithClient(gr.Manager.GetClient()).WithScheme(gr.Manager.GetScheme())
	gr.RsrcMgr.Add(k8s.Type, km)
	app.AddToScheme(&AddToSchemes)
	AddToSchemes.AddToScheme(gr.Manager.GetScheme())
}
