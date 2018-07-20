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

package common

import (
	"fmt"
	airflowv1alpha1 "k8s.io/airflow-operator/pkg/apis/airflow/v1alpha1"
	resources "k8s.io/airflow-operator/pkg/controller/resources"
	"k8s.io/airflow-operator/pkg/inject/args"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	urt "k8s.io/apimachinery/pkg/util/runtime"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	policyinformers "k8s.io/client-go/informers/policy/v1beta1"
	rbacinformers "k8s.io/client-go/informers/rbac/v1"
	"k8s.io/client-go/kubernetes"
	appslister "k8s.io/client-go/listers/apps/v1"
	corelister "k8s.io/client-go/listers/core/v1"
	policylister "k8s.io/client-go/listers/policy/v1beta1"
	rbaclister "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/record"
	"log"
	"reflect"
)

// Controller defines fields needed for all airflow controllers
type Controller struct {
	// Kubernetes clientset to access core resources
	K8sclientset *kubernetes.Clientset
	// Listers
	StsLister    appslister.StatefulSetLister
	SvcLister    corelister.ServiceLister
	PvcLister    corelister.PersistentVolumeClaimLister
	SecretLister corelister.SecretLister
	PdbLister    policylister.PodDisruptionBudgetLister
	SALister     corelister.ServiceAccountLister
	RBLister     rbaclister.RoleBindingLister
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

func handleErrorArr(info string, name string, e error, errs []error) []error {
	HandleError(info, name, e)
	return append(errs, e)
}

// HandleError common error handling routine
func HandleError(info string, name string, e error) error {
	urt.HandleError(fmt.Errorf("Failed: [%s] %s. %s", name, info, e.Error()))
	return e
}

// Get is used to retrieve a single object
func (c *Controller) list(selector airflowv1alpha1.ResourceSelector) ([]airflowv1alpha1.ResourceInfo, error) {
	var rs []airflowv1alpha1.ResourceInfo

	switch selector.Obj.(type) {
	case *resources.StatefulSet:
		objs, err := c.StsLister.List(selector.Selectors)
		if err != nil {
			return nil, err
		}
		for _, obj := range objs {
			rs = append(rs, airflowv1alpha1.ResourceInfo{Lifecycle: "", Obj: &resources.StatefulSet{StatefulSet: obj}, Action: ""})
		}
	case *resources.Service:
		objs, err := c.SvcLister.List(selector.Selectors)
		if err != nil {
			return nil, err
		}
		for _, obj := range objs {
			rs = append(rs, airflowv1alpha1.ResourceInfo{Lifecycle: "", Obj: &resources.Service{Service: obj}, Action: ""})
		}
	case *resources.PodDisruptionBudget:
		objs, err := c.PdbLister.List(selector.Selectors)
		if err != nil {
			return nil, err
		}
		for _, obj := range objs {
			rs = append(rs, airflowv1alpha1.ResourceInfo{Lifecycle: "", Obj: &resources.PodDisruptionBudget{PodDisruptionBudget: obj}, Action: ""})
		}
	case *resources.Secret:
		objs, err := c.SecretLister.List(selector.Selectors)
		if err != nil {
			return nil, err
		}
		for _, obj := range objs {
			rs = append(rs, airflowv1alpha1.ResourceInfo{Lifecycle: "", Obj: &resources.Secret{Secret: obj}, Action: ""})
		}
	case *resources.ServiceAccount:
		objs, err := c.SALister.List(selector.Selectors)
		if err != nil {
			return nil, err
		}
		for _, obj := range objs {
			rs = append(rs, airflowv1alpha1.ResourceInfo{Lifecycle: "", Obj: &resources.ServiceAccount{ServiceAccount: obj}, Action: ""})
		}
	case *resources.RoleBinding:
		objs, err := c.RBLister.List(selector.Selectors)
		if err != nil {
			return nil, err
		}
		for _, obj := range objs {
			rs = append(rs, airflowv1alpha1.ResourceInfo{Lifecycle: "", Obj: &resources.RoleBinding{RoleBinding: obj}, Action: ""})
		}
	}
	return rs, nil
}

func (c *Controller) observe(selectors []airflowv1alpha1.ResourceSelector) ([]airflowv1alpha1.ResourceInfo, error) {
	var resources, returnval []airflowv1alpha1.ResourceInfo
	var err error
	for _, selector := range selectors {
		if selector.Selectors != nil {
			resources, err = c.list(selector)
		} else {
			obj := selector.Obj
			cname := reflect.TypeOf(obj).String() + "/" + obj.GetNamespace() + "/" + obj.GetName()
			obj, err = obj.Get(c.K8sclientset)
			resources = []airflowv1alpha1.ResourceInfo{{
				Lifecycle: "",
				Obj:       obj,
				Action:    ""}}
			if err == nil {
				log.Printf("   >get: %s", cname)
			}
		}
		if err != nil {
			return nil, err
		}
		for _, resource := range resources {
			returnval = append(returnval, resource)
		}
	}
	return returnval, nil
}

// Reconcile is a generic function that reconciles expected and observed resources
// This is called per component for the components of AirflowBase and AirflowCluster
func (c *Controller) Reconcile(cname string, rname string, expected []airflowv1alpha1.ResourceInfo, selectors []airflowv1alpha1.ResourceSelector, component airflowv1alpha1.ComponentHandle, status interface{}) {
	errs := []error{}
	reconciled := []airflowv1alpha1.ResourceInfo{}

	cname = rname + "/" + cname
	log.Printf("%s  { reconciling component\n", cname)
	defer log.Printf("%s  } reconciling component\n", cname)
	observed, err := c.observe(selectors)
	if err != nil {
		HandleError("observing resources", rname, err)
		component.UpdateStatus(status, reconciled, err)
		return
	}

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

	log.Printf("%s  Expected Resources:\n", cname)
	for _, e := range expected {
		log.Printf("%s   exp: %s/%s/%s\n", cname, e.Obj.GetNamespace(), reflect.TypeOf(e.Obj).String(), e.Obj.GetName())
	}
	log.Printf("%s  Observed Resources:\n", cname)
	for _, e := range observed {
		log.Printf("%s   obs: %s/%s/%s\n", cname, e.Obj.GetNamespace(), reflect.TypeOf(e.Obj).String(), e.Obj.GetName())
	}

	log.Printf("%s  Reconciling Resources:\n", cname)
	for _, e := range expected {
		seen := false
		eNamespace := e.Obj.GetNamespace()
		eName := e.Obj.GetName()
		eKind := reflect.TypeOf(e.Obj).String()
		eRsrcInfo := eKind + "/" + eNamespace + "/" + eName
		for _, o := range observed {
			if (eName == o.Obj.GetName()) &&
				(eNamespace == o.Obj.GetNamespace()) &&
				(eKind == reflect.TypeOf(o.Obj).String()) {
				// rsrc is seen in both expected and observed, update it if needed
				if e.Lifecycle == airflowv1alpha1.LifecycleManaged && component.Differs(e, o) {
					if err := e.Obj.Update(c.K8sclientset); err != nil {
						errs = handleErrorArr("update", eRsrcInfo, err, errs)
					} else {
						log.Printf("%s   *update: %s\n", cname, eRsrcInfo)
					}
				}
				reconciled = append(reconciled, o)
				seen = true
				break
			}
		}
		// rsrc is in expected but not in observed - create
		if !seen {
			if e.Lifecycle == airflowv1alpha1.LifecycleManaged {
				if o, err := e.Obj.Create(c.K8sclientset); err != nil {
					errs = handleErrorArr("Create", cname, err, errs)
				} else {
					log.Printf("%s   +create: %s\n", cname, eRsrcInfo)
					e.Obj = o
					reconciled = append(reconciled, e)
				}
			} else {
				err := fmt.Errorf("missing resource not managed by %s: %s", cname, eRsrcInfo)
				errs = handleErrorArr("missing resource", cname, err, errs)
			}
		}
	}

	// delete(observed - expected)
	for _, o := range observed {
		seen := false
		oNamespace := o.Obj.GetNamespace()
		oName := o.Obj.GetName()
		oKind := reflect.TypeOf(o.Obj).String()
		oRsrcInfo := oKind + "/" + oNamespace + "/" + oName
		for _, e := range expected {
			if (e.Obj.GetName() == oName) &&
				(e.Obj.GetNamespace() == oNamespace) &&
				(reflect.TypeOf(o.Obj).String() == oKind) {
				seen = true
				break
			}
		}
		// rsrc is in observed but not in expected - delete
		if !seen {
			if err := o.Obj.Delete(c.K8sclientset); err != nil {
				errs = handleErrorArr("delete", oRsrcInfo, err, errs)
			} else {
				log.Printf("%s   -delete: %s\n", cname, oRsrcInfo)
			}
		}
	}

	component.UpdateStatus(status, reconciled, utilerrors.NewAggregate(errs))
}

// ProvideController is used to return a populated controller object
func ProvideController(arguments args.InjectArgs, controllerName string) Controller {
	return Controller{
		K8sclientset: arguments.KubernetesClientSet,
		StsLister: arguments.ControllerManager.GetInformerProvider(
			&appsv1.StatefulSet{}).(appsinformers.StatefulSetInformer).Lister(),
		SvcLister: arguments.ControllerManager.GetInformerProvider(
			&corev1.Service{}).(coreinformers.ServiceInformer).Lister(),
		PvcLister: arguments.ControllerManager.GetInformerProvider(
			&corev1.PersistentVolumeClaim{}).(coreinformers.PersistentVolumeClaimInformer).Lister(),
		SecretLister: arguments.ControllerManager.GetInformerProvider(
			&corev1.Secret{}).(coreinformers.SecretInformer).Lister(),
		SALister: arguments.ControllerManager.GetInformerProvider(
			&corev1.ServiceAccount{}).(coreinformers.ServiceAccountInformer).Lister(),
		RBLister: arguments.ControllerManager.GetInformerProvider(
			&rbacv1.RoleBinding{}).(rbacinformers.RoleBindingInformer).Lister(),
		PdbLister: arguments.ControllerManager.GetInformerProvider(
			&policyv1.PodDisruptionBudget{}).(policyinformers.PodDisruptionBudgetInformer).Lister(),
		recorder: arguments.CreateRecorder(controllerName),
	}
}
