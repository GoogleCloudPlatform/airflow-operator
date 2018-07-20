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

package airflowcluster

import (
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/eventhandlers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	airflowv1alpha1 "k8s.io/airflow-operator/pkg/apis/airflow/v1alpha1"
	airflowv1alpha1client "k8s.io/airflow-operator/pkg/client/clientset/versioned/typed/airflow/v1alpha1"
	airflowv1alpha1informer "k8s.io/airflow-operator/pkg/client/informers/externalversions/airflow/v1alpha1"
	airflowv1alpha1lister "k8s.io/airflow-operator/pkg/client/listers/airflow/v1alpha1"
	common "k8s.io/airflow-operator/pkg/controller/airflowcommon"
	resources "k8s.io/airflow-operator/pkg/controller/resources"
	"k8s.io/airflow-operator/pkg/inject/args"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
)

func (c *ACController) update(name, namespace string, rsrc *airflowv1alpha1.AirflowCluster) error {
	_, err := c.airflowclusterclient.AirflowClusters(namespace).Update(rsrc)
	if err != nil {
		return common.HandleError("Update Status ", name, err)
	}
	return nil
}

// Reconcile gets the Resource and reconciles it with real world state
func (c *ACController) Reconcile(k types.ReconcileKey) error {
	name := "AC/" + k.String()
	log.Printf("%s { Start Reconcile\n", name)
	defer log.Printf("%s } End Reconcile\n", name)
	rsrc, err := c.airflowclusterclient.AirflowClusters(k.Namespace).Get(k.Name, metav1.GetOptions{})
	if err != nil || rsrc == nil {
		if apierrors.IsNotFound(err) {
			common.HandleError("Airflow Resource Notfound", name, err)
			return nil
		}
		return common.HandleError("NotFound", name, err)
	}

	if rsrc.DeletionTimestamp != nil {
		log.Printf("%s Deletion in progress\n", name)
		return nil
	}

	newStatus := airflowv1alpha1.AirflowClusterStatus{}
	newStatus.ObservedGeneration = rsrc.Generation
	newStatus.Status = airflowv1alpha1.StatusReady
	log.Printf("%s Applying defaults\n", name)
	rsrc.ApplyDefaults()
	log.Printf("%s Validating spec\n", name)
	if err = rsrc.Validate(); err != nil {
		common.HandleError("Validate", name, err)
		rsrc.Status.LastError = err.Error()
		c.update(name, k.Namespace, rsrc)
		return err
	}

	components := rsrc.Components()
	for cname, cobject := range components {
		component := cobject.(airflowv1alpha1.ComponentHandle)
		expected := component.ExpectedResources(rsrc)
		selectors := component.ObserveSelectors(rsrc)
		c.Controller.Reconcile(cname, name, expected, selectors, component, &newStatus)
	}

	if rsrc.StatusDiffers(newStatus) {
		rsrc.Status = newStatus
		c.update(name, k.Namespace, rsrc)
	}
	return nil
}

// ACController represents the controller
// +kubebuilder:controller:group=airflow,version=v1alpha1,kind=AirflowCluster,resource=airflowclusters
// +kubebuilder:informers:group=apps,version=v1,kind=StatefulSet
// +kubebuilder:informers:group=core,version=v1,kind=ConfigMap
// +kubebuilder:informers:group=core,version=v1,kind=PersistentVolumeClaim
// +kubebuilder:informers:group=core,version=v1,kind=Pod
// +kubebuilder:informers:group=core,version=v1,kind=Secret
// +kubebuilder:informers:group=core,version=v1,kind=Service
// +kubebuilder:informers:group=policy,version=v1beta1,kind=PodDisruptionBudget
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;watch;list
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;watch;list
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch;create;update;patch;delete
type ACController struct {
	// INSERT ADDITIONAL FIELDS HERE
	airflowclusterLister airflowv1alpha1lister.AirflowClusterLister
	airflowclusterclient airflowv1alpha1client.AirflowV1alpha1Interface
	// common controller interface
	common.Controller
}

func (c *ACController) stsToAirflowCluster(k types.ReconcileKey) (interface{}, error) {
	sts, err := (&resources.StatefulSet{StatefulSet: &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: k.Name, Namespace: k.Namespace}}}).Get(c.Controller.K8sclientset)
	if err != nil {
		return nil, err
	}
	owners := sts.GetOwnerReferences()

	if owners == nil || owners[0].Kind != airflowv1alpha1.KindAirflowCluster {
		return nil, nil
	}

	rsrc, err := c.airflowclusterclient.AirflowClusters(k.Namespace).Get(owners[0].Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return rsrc, nil
}

// ProvideController provides a controller that will be run at startup.  Kubebuilder will use codegeneration
// to automatically register this controller in the inject package
func ProvideController(arguments args.InjectArgs) (*controller.GenericController, error) {
	bc := &ACController{
		airflowclusterLister: arguments.ControllerManager.GetInformerProvider(&airflowv1alpha1.AirflowCluster{}).(airflowv1alpha1informer.AirflowClusterInformer).Lister(),
		airflowclusterclient: arguments.Clientset.AirflowV1alpha1(),
		Controller:           common.ProvideController(arguments, "AirflowCluster-Controller"),
	}

	// Create a new controller that will call AirflowClusterController.Reconcile on changes to AirflowClusters
	gc := &controller.GenericController{
		Name:             "AirflowClusterController",
		Reconcile:        bc.Reconcile,
		InformerRegistry: arguments.ControllerManager,
	}
	if err := gc.Watch(&airflowv1alpha1.AirflowCluster{}); err != nil {
		return gc, err
	}

	// ADDITIONAL WATCHES
	// Informers for k8s resources registered in the pkg/inject package so that they are started.
	if err := gc.WatchControllerOf(&appsv1.StatefulSet{},
		eventhandlers.Path{bc.stsToAirflowCluster}); err != nil {
		return gc, err
	}
	return gc, nil
}
