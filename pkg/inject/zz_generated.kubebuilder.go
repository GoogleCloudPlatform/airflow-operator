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
package inject

import (
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	airflowv1alpha1 "k8s.io/airflow-operator/pkg/apis/airflow/v1alpha1"
	rscheme "k8s.io/airflow-operator/pkg/client/clientset/versioned/scheme"
	"k8s.io/airflow-operator/pkg/controller/airflowbase"
	"k8s.io/airflow-operator/pkg/controller/airflowcluster"
	"k8s.io/airflow-operator/pkg/inject/args"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	rscheme.AddToScheme(scheme.Scheme)

	// Inject Informers
	Inject = append(Inject, func(arguments args.InjectArgs) error {
		Injector.ControllerManager = arguments.ControllerManager

		if err := arguments.ControllerManager.AddInformerProvider(&airflowv1alpha1.AirflowBase{}, arguments.Informers.Airflow().V1alpha1().AirflowBases()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&airflowv1alpha1.AirflowCluster{}, arguments.Informers.Airflow().V1alpha1().AirflowClusters()); err != nil {
			return err
		}

		// Add Kubernetes informers
		if err := arguments.ControllerManager.AddInformerProvider(&corev1.Service{}, arguments.KubernetesInformers.Core().V1().Services()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&policyv1beta1.PodDisruptionBudget{}, arguments.KubernetesInformers.Policy().V1beta1().PodDisruptionBudgets()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&appsv1.StatefulSet{}, arguments.KubernetesInformers.Apps().V1().StatefulSets()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1.ConfigMap{}, arguments.KubernetesInformers.Core().V1().ConfigMaps()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1.PersistentVolumeClaim{}, arguments.KubernetesInformers.Core().V1().PersistentVolumeClaims()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1.Pod{}, arguments.KubernetesInformers.Core().V1().Pods()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1.Secret{}, arguments.KubernetesInformers.Core().V1().Secrets()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&corev1.ServiceAccount{}, arguments.KubernetesInformers.Core().V1().ServiceAccounts()); err != nil {
			return err
		}
		if err := arguments.ControllerManager.AddInformerProvider(&rbacv1.RoleBinding{}, arguments.KubernetesInformers.Rbac().V1().RoleBindings()); err != nil {
			return err
		}

		if c, err := airflowbase.ProvideController(arguments); err != nil {
			return err
		} else {
			arguments.ControllerManager.AddController(c)
		}
		if c, err := airflowcluster.ProvideController(arguments); err != nil {
			return err
		} else {
			arguments.ControllerManager.AddController(c)
		}
		return nil
	})

	// Inject CRDs
	Injector.CRDs = append(Injector.CRDs, &airflowv1alpha1.AirflowBaseCRD)
	Injector.CRDs = append(Injector.CRDs, &airflowv1alpha1.AirflowClusterCRD)
	// Inject PolicyRules
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{"airflow.k8s.io"},
		Resources: []string{"*"},
		Verbs:     []string{"*"},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{
			"",
		},
		Resources: []string{
			"persistentvolumeclaims",
		},
		Verbs: []string{
			"get", "list", "watch",
		},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{
			"",
		},
		Resources: []string{
			"pods",
		},
		Verbs: []string{
			"get", "list", "watch",
		},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{
			"core",
		},
		Resources: []string{
			"secrets",
		},
		Verbs: []string{
			"create", "delete", "get", "list", "patch", "update", "watch",
		},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{
			"core",
		},
		Resources: []string{
			"services",
		},
		Verbs: []string{
			"create", "delete", "get", "list", "patch", "update", "watch",
		},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{
			"policy",
		},
		Resources: []string{
			"poddisruptionbudgets",
		},
		Verbs: []string{
			"create", "delete", "get", "list", "patch", "update", "watch",
		},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{
			"apps",
		},
		Resources: []string{
			"statefulsets",
		},
		Verbs: []string{
			"create", "delete", "get", "list", "patch", "update", "watch",
		},
	})
	Injector.PolicyRules = append(Injector.PolicyRules, rbacv1.PolicyRule{
		APIGroups: []string{
			"core",
		},
		Resources: []string{
			"configmaps",
		},
		Verbs: []string{
			"create", "delete", "get", "list", "patch", "update", "watch",
		},
	})
	// Inject GroupVersions
	Injector.GroupVersions = append(Injector.GroupVersions, schema.GroupVersion{
		Group:   "airflow.k8s.io",
		Version: "v1alpha1",
	})
	Injector.RunFns = append(Injector.RunFns, func(arguments run.RunArguments) error {
		Injector.ControllerManager.RunInformersAndControllers(arguments)
		return nil
	})
}
