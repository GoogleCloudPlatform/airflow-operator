/*
Copyright 2018 Google LLC.

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
	"context"
	airflowv1alpha1 "k8s.io/airflow-operator/pkg/apis/airflow/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	reconciler "sigs.k8s.io/kubesdk/pkg/genericreconciler"
	kbc "sigs.k8s.io/kubesdk/pkg/kbcontroller"
	"sigs.k8s.io/kubesdk/pkg/resource/manager/gcp/redis"
)

// Add creates a new AirflowCluster Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return kbc.CreateController("airflowcluster", mgr, &airflowv1alpha1.AirflowCluster{}, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	r := &reconciler.Reconciler{}
	r.WithManager(mgr).WithCR(&airflowv1alpha1.AirflowCluster{}).Init()
	if rd, err := redis.NewRsrcManager(context.TODO(), "redismgr"); err == nil {
		r.RsrcMgr.Add(redis.Type, rd)
	}
	return r
}
