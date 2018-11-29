/*
Copyright 2018 The Kubernetes Authors.
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

// Package e2e handles e2e tests for airflow operator
package e2e

import (
	"context"
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/airflow-operator/pkg/apis"
	airflowv1alpha1 "k8s.io/airflow-operator/pkg/apis/airflow/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/kubesdk/pkg/component"
	"sync"
	//need blank import
	"k8s.io/apimachinery/pkg/util/wait"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"os"
	"testing"
	"time"
)

const (
	operatorsts            = "airflow-operator-test"
	namespace              = "aotest"
	pollinterval           = 1 * time.Second
	ControllerImageEnvName = "CONTROLLER_IMAGE"
)

var k8sClient client.Client

func operatorLabels() map[string]string {
	return map[string]string{
		"api":           "airflow",
		"control-plane": "controller-manager",
		"platform":      "kubernetes_sigs_kubebuilder",
	}
}

func operatorNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespace,
			Namespace: namespace,
			Labels:    operatorLabels(),
		},
	}
}

func operatorCRB() *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespace + "crb",
			Namespace: namespace,
			Labels:    operatorLabels(),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: namespace,
			},
		},
	}
}

func operatorSts() *appsv1.StatefulSet {
	controllerImage := os.Getenv(ControllerImageEnvName)
	if controllerImage == "" {
		controllerImage = "gcr.io/airflow-operator/airflow-operator:v1alpha1"
	}
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorsts,
			Namespace: namespace,
			Labels:    operatorLabels(),
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: operatorLabels()},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: operatorLabels()},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "controller-manager",
							Args:    []string{"--install-crds=true"},
							Command: []string{"/root/controller-manager"},
							Image:   controllerImage,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("20Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("30Mi"),
								},
							},
						},
					},
				},
			},
		},
	}
}

func airflowBase(test, database string) *airflowv1alpha1.AirflowBase {
	ab := airflowv1alpha1.NewAirflowBase(test, namespace, database, true)
	ab.ObjectMeta.Labels["createdby"] = "e2etest"
	return ab
}

func airflowCluster(test, base, executor string, dags *airflowv1alpha1.DagSpec) *airflowv1alpha1.AirflowCluster {
	ac := airflowv1alpha1.NewAirflowCluster(test, namespace, executor, base, dags)
	ac.ObjectMeta.Labels["createdby"] = "e2etest"
	ac.ObjectMeta.Labels["test"] = test
	return ac
}

func waitAirflowCluster(ac *airflowv1alpha1.AirflowCluster) {
	testname := ac.GetName()
	executor := ac.Spec.Executor
	waitForSecret(ac, testname, airflowv1alpha1.ValueAirflowComponentUI)
	waitSts(ac, testname, airflowv1alpha1.ValueAirflowCRCluster, airflowv1alpha1.ValueAirflowComponentScheduler)
	waitSts(ac, testname, airflowv1alpha1.ValueAirflowCRCluster, airflowv1alpha1.ValueAirflowComponentUI)
	if executor == airflowv1alpha1.ExecutorCelery {
		waitSts(ac, testname, airflowv1alpha1.ValueAirflowCRCluster, airflowv1alpha1.ValueAirflowComponentWorker)
		waitSts(ac, testname, airflowv1alpha1.ValueAirflowCRCluster, airflowv1alpha1.ValueAirflowComponentRedis)
	}

	var acs *airflowv1alpha1.AirflowClusterList
	var err error
	listop := client.MatchingLabels(labels.Set(map[string]string{"test": testname}))
	err = wait.PollImmediate(pollinterval, 2*time.Minute, func() (bool, error) {
		err = k8sClient.List(context.TODO(), listop, acs)
		if err != nil {
			return false, fmt.Errorf("failed to list airflowcluster: %s, %v", testname, err)
		}
		return len(acs.Items) == 1, nil
	})
	Expect(err).NotTo(HaveOccurred(), "error waiting for airflowcluster sts: %s, %v", testname, err)

	listedac := &acs.Items[0]
	err = wait.PollImmediate(pollinterval, 3*time.Minute, func() (bool, error) {
		err = k8sClient.Get(context.TODO(),
			types.NamespacedName{Name: listedac.Name, Namespace: listedac.Namespace},
			listedac)
		if err != nil {
			return false, fmt.Errorf("failed to get airflowcluster: %s, %v", testname, err)
		}
		return listedac.Status.IsReady(), nil
	})
	Expect(err).NotTo(HaveOccurred(), "error waiting for airfloewcluster to be ready: %s, %v", testname, err)

	return
}

func waitAirflowBase(ab *airflowv1alpha1.AirflowBase, database string) {
	testname := ab.GetName()

	if database == airflowv1alpha1.DatabaseMySQL {
		waitForSecret(ab, testname, airflowv1alpha1.ValueAirflowComponentSQL)
		waitSts(ab, testname, airflowv1alpha1.ValueAirflowCRBase, airflowv1alpha1.ValueAirflowComponentMySQL)
	} else if database == airflowv1alpha1.DatabasePostgres {
		waitSts(ab, testname, airflowv1alpha1.ValueAirflowCRBase, airflowv1alpha1.ValueAirflowComponentPostgres)
	} else {
		waitSts(ab, testname, airflowv1alpha1.ValueAirflowCRBase, airflowv1alpha1.ValueAirflowComponentSQLProxy)
	}
	waitSts(ab, testname, airflowv1alpha1.ValueAirflowCRBase, airflowv1alpha1.ValueAirflowComponentNFS)

	var airflowbases *airflowv1alpha1.AirflowBaseList
	var err error
	listop := client.MatchingLabels(labels.Set(map[string]string{"test": testname}))
	err = wait.PollImmediate(pollinterval, 2*time.Minute, func() (bool, error) {
		err = k8sClient.List(context.TODO(), listop, airflowbases)
		if err != nil {
			return false, fmt.Errorf("failed to list airflowbase: %s, %v", testname, err)
		}
		return len(airflowbases.Items) == 1, nil
	})
	Expect(err).NotTo(HaveOccurred(), "error waiting for airflowbase sts: %s, %v", testname, err)

	listedab := &airflowbases.Items[0]
	err = wait.PollImmediate(pollinterval, 3*time.Minute, func() (bool, error) {
		err = k8sClient.Get(context.TODO(),
			types.NamespacedName{Name: listedab.Name, Namespace: listedab.Namespace},
			listedab)
		if err != nil {
			return false, fmt.Errorf("failed to get airflowbase: %s, %v", testname, err)
		}
		return listedab.Status.IsReady(), nil
	})
	Expect(err).NotTo(HaveOccurred(), "error waiting for airfloewbase to be ready: %s, %v", testname, err)

	return
}

func waitSts(obj metav1.Object, testname, rsrc, cname string) {
	var stss *appsv1.StatefulSetList
	var err error
	selector := component.Labels(obj, cname)
	listop := client.MatchingLabels(labels.Set(selector))
	err = wait.PollImmediate(pollinterval, 2*time.Minute, func() (bool, error) {
		err = k8sClient.List(context.TODO(), listop, stss)
		if err != nil {
			return false, fmt.Errorf("failed to list sts: %s, %v", cname, err)
		}
		return len(stss.Items) == 1, nil
	})
	Expect(err).NotTo(HaveOccurred(), "error waiting for list sts: %s, %v", cname, err)

	sts := &stss.Items[0]
	err = wait.PollImmediate(pollinterval, 2*time.Minute, func() (bool, error) {
		err = k8sClient.Get(context.TODO(),
			types.NamespacedName{Name: sts.Name, Namespace: sts.Namespace},
			sts)
		if err != nil {
			fmt.Printf("error getting sts (ns: %s) %s\n", sts.Namespace, err.Error())
			return false, fmt.Errorf("failed to get sts: %s, %v", cname, err)
		}
		return sts.Status.ReadyReplicas == *sts.Spec.Replicas && sts.Status.CurrentReplicas == *sts.Spec.Replicas, nil
	})
	Expect(err).NotTo(HaveOccurred(), "error waiting for sts to be ready: %s, %v", cname, err)

	return
}

func waitForSecret(obj metav1.Object, name, cname string) {
	var err error
	var secret = corev1.Secret{}
	err = wait.PollImmediate(pollinterval, 1*time.Minute, func() (bool, error) {
		err = k8sClient.Get(context.TODO(),
			types.NamespacedName{Name: name + "-" + cname, Namespace: namespace},
			&secret)
		if err != nil {
			fmt.Printf("error getting secret (ns: %s) %s\n", namespace, err.Error())
			return false, nil
		}
		return true, nil
	})
	Expect(err).NotTo(HaveOccurred(), "error waiting for secret: %s, %v", cname, err)

	return
}

// TestE2e entry point ?
func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Application Type Suite", []Reporter{})
}

// BeforeSuite
// AfterSuite
// Describe
//  BeforeEach
//  AfterEach
//  It
//   Expect

func InitClient() {
	apis.AddToScheme(scheme.Scheme)
	cfg, err := config.GetConfig()
	if err != nil {
		os.Exit(1)
	}
	mgr, err := manager.New(cfg, manager.Options{})
	Expect(err).NotTo(HaveOccurred())
	k8sClient = mgr.GetClient()
	stop := make(chan struct{})
	wg := &sync.WaitGroup{}

	go func() {
		wg.Add(1)
		Expect(mgr.Start(stop)).NotTo(HaveOccurred())
		wg.Done()
	}()

	defer func() {
		close(stop)
		wg.Wait()
	}()
}

func getDeleteOptions() *metav1.DeleteOptions {
	deletePropagationBackgroundPolicy := metav1.DeletePropagationBackground
	zero := int64(0)
	return &metav1.DeleteOptions{
		PropagationPolicy:  &deletePropagationBackgroundPolicy,
		GracePeriodSeconds: &zero,
	}
}

var _ = BeforeSuite(func() {
	InitClient()
	if false {
		By("Creating airflow controller statefulset")
		err := k8sClient.Create(context.TODO(), operatorNamespace())
		Expect(err).NotTo(HaveOccurred(), "failed to create test namespace: %v", err)
		err = k8sClient.Create(context.TODO(), operatorCRB())
		Expect(err).NotTo(HaveOccurred(), "failed to create operator CRB: %v", err)
		err = k8sClient.Create(context.TODO(), operatorSts())
		Expect(err).NotTo(HaveOccurred(), "failed to create operator statefulset %q: %v", operatorsts, err)
	}
})

var _ = AfterSuite(func() {
	if false {
		By("Deleting airflow controller statefulset")
		err := k8sClient.Delete(context.TODO(), operatorSts())
		Expect(err).NotTo(HaveOccurred(), "failed to delete operator statefulset %q: %v", operatorsts, err)
		err = k8sClient.Delete(context.TODO(), operatorCRB())
		Expect(err).NotTo(HaveOccurred(), "failed to delete operator CRB: %v", err)
		err = k8sClient.Delete(context.TODO(), operatorSts())
		Expect(err).NotTo(HaveOccurred(), "failed to delete test namespace: %v", err)
	}
})

var _ = Describe("AirflowBase and AirflowCluster Deployment should work", func() {
	var basem string

	BeforeEach(func() {})
	AfterEach(func() {})

	// --------------- Tests -------------------------
	It("should create airflow-base using mysql and nfs components", func() {
		basem = testBaseComponentCreation(airflowv1alpha1.DatabaseMySQL)
	})

	// airflowBase() needs to inject sqlproxy config
	//It("should create airflow-base using sqlproxy and nfs components", func() { testBaseComponentCreation(false) })

	dags := &airflowv1alpha1.DagSpec{
		DagSubdir: "airflow/example_dags/",
		Git: &airflowv1alpha1.GitSpec{
			Repo: "https://github.com/apache/incubator-airflow/",
			Once: true,
		},
	}
	It("should create airflow-cluster using celery,git and using mysql base", func() {
		testClusterComponentCreation(basem, airflowv1alpha1.ExecutorCelery, airflowv1alpha1.DatabaseMySQL, dags)
	})

})

// Step 1: create AirflowBase object with storage and mysql/sqlproxy
// Step 2: wait for mysql/sqlproxy StatefulSet and nfs StatefulSet to become ready all pods have to be available
// Step 3: verify for mysql, the root password secret should be created, Mysql and nfs stateful sets should have 1 pods each
func testBaseComponentCreation(database string) string {
	testname := "base"
	if database == airflowv1alpha1.DatabaseMySQL {
		testname += "-m"
	} else if database == airflowv1alpha1.DatabasePostgres {
		testname += "-p"
	} else {
		testname += "-s"
	}
	ab := airflowBase(testname, database)
	err := k8sClient.Create(context.TODO(), ab)
	Expect(err).NotTo(HaveOccurred(), "failed to create AirflowBase %q: %v", testname, err)

	waitAirflowBase(ab, database)

	return testname
}

//Step 3: create AirflowCluster object with redis, ui, scheduler and workers enabled and celery executor
//Step 4: wait for redis, ui, scheduler and worker StatefulSets to become ready, all pods have to be available
//Step 5: verify
//     All stateful sets have 1 pod each
//     Scheduler is configured correctly to connect to mysql, and celery connection string points to redis instance
//     UI is configured correctly to connect to mysql
//     Workers celery config and mysql config is correct
//     Scheduler, ui and workers all synced the DAGs from git repo
func testClusterComponentCreation(base, executor, database string, dags *airflowv1alpha1.DagSpec) {
	testname := "cluster-cmg"
	ac := airflowCluster(testname, base, executor, dags)
	err := k8sClient.Create(context.TODO(), ac)
	Expect(err).NotTo(HaveOccurred(), "failed to create AirflowCluster %q: %v", testname, err)

	By(fmt.Sprintf("verifying AirflowCluster %s components are created", testname))
	waitAirflowCluster(ac)
}
