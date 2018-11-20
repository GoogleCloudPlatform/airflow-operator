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
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	airflowv1alpha1 "k8s.io/airflow-operator/pkg/apis/airflow/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	//need blank import
	typedairflow "k8s.io/airflow-operator/pkg/client/clientset/versioned/typed/airflow/v1alpha1"
	"k8s.io/apimachinery/pkg/util/wait"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"path"
	"testing"
	"time"
)

const (
	operatorsts            = "airflow-operator-test"
	namespace              = "aotest"
	pollinterval           = 1 * time.Second
	ControllerImageEnvName = "CONTROLLER_IMAGE"
)

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

func airflowBase(test string, mysql bool) *airflowv1alpha1.AirflowBase {
	ab := airflowv1alpha1.NewAirflowBase(test, namespace, mysql, true)
	ab.ObjectMeta.Labels["createdby"] = "e2etest"
	return ab
}

func airflowCluster(test, base, executor string, dags *airflowv1alpha1.DagSpec) *airflowv1alpha1.AirflowCluster {
	ac := airflowv1alpha1.NewAirflowCluster(test, namespace, executor, base, dags)
	ac.ObjectMeta.Labels["createdby"] = "e2etest"
	ac.ObjectMeta.Labels["test"] = test
	return ac
}

func setToListSelector(set map[string]string) metav1.ListOptions {
	return metav1.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set(set)).String()}
}

func waitAirflowCluster(k8sclient *kubernetes.Clientset, aclient *typedairflow.AirflowV1alpha1Client, testname, executor string) {
	waitForSecret(k8sclient, testname, airflowv1alpha1.ValueAirflowComponentUI)
	waitSts(k8sclient, testname, airflowv1alpha1.ValueAirflowCRCluster, airflowv1alpha1.ValueAirflowComponentScheduler)
	waitSts(k8sclient, testname, airflowv1alpha1.ValueAirflowCRCluster, airflowv1alpha1.ValueAirflowComponentUI)
	if executor == airflowv1alpha1.ExecutorCelery {
		waitSts(k8sclient, testname, airflowv1alpha1.ValueAirflowCRCluster, airflowv1alpha1.ValueAirflowComponentWorker)
		waitSts(k8sclient, testname, airflowv1alpha1.ValueAirflowCRCluster, airflowv1alpha1.ValueAirflowComponentRedis)
	}

	var acs *airflowv1alpha1.AirflowClusterList
	var err error
	acclient := aclient.AirflowClusters(namespace)
	listop := metav1.ListOptions{LabelSelector: labels.Set(map[string]string{"test": testname}).AsSelector().String()}
	err = wait.PollImmediate(pollinterval, 2*time.Minute, func() (bool, error) {
		acs, err = acclient.List(listop)
		if err != nil {
			return false, fmt.Errorf("failed to list airflowcluster: %s, %v", testname, err)
		}
		return len(acs.Items) == 1, nil
	})
	Expect(err).NotTo(HaveOccurred(), "error waiting for airflowcluster sts: %s, %v", testname, err)

	ac := &acs.Items[0]
	err = wait.PollImmediate(pollinterval, 3*time.Minute, func() (bool, error) {
		ac, err = acclient.Get(ac.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get airflowcluster: %s, %v", testname, err)
		}
		return ac.Status.Status == airflowv1alpha1.StatusReady, nil
	})
	Expect(err).NotTo(HaveOccurred(), "error waiting for airfloewcluster to be ready: %s, %v", testname, err)

	return
}

func waitAirflowBase(k8sclient *kubernetes.Clientset, aclient *typedairflow.AirflowV1alpha1Client, testname string, mysql bool) {
	if mysql {
		waitForSecret(k8sclient, testname, airflowv1alpha1.ValueAirflowComponentSQL)
		waitSts(k8sclient, testname, airflowv1alpha1.ValueAirflowCRBase, airflowv1alpha1.ValueAirflowComponentMySQL)
	} else {
		waitSts(k8sclient, testname, airflowv1alpha1.ValueAirflowCRBase, airflowv1alpha1.ValueAirflowComponentSQLProxy)
	}
	waitSts(k8sclient, testname, airflowv1alpha1.ValueAirflowCRBase, airflowv1alpha1.ValueAirflowComponentNFS)

	var airflowbases *airflowv1alpha1.AirflowBaseList
	var err error
	abclient := aclient.AirflowBases(namespace)
	listop := metav1.ListOptions{LabelSelector: labels.Set(map[string]string{"test": testname}).AsSelector().String()}
	err = wait.PollImmediate(pollinterval, 2*time.Minute, func() (bool, error) {
		airflowbases, err = abclient.List(listop)
		if err != nil {
			return false, fmt.Errorf("failed to list airflowbase: %s, %v", testname, err)
		}
		return len(airflowbases.Items) == 1, nil
	})
	Expect(err).NotTo(HaveOccurred(), "error waiting for airflowbase sts: %s, %v", testname, err)

	ab := &airflowbases.Items[0]
	err = wait.PollImmediate(pollinterval, 3*time.Minute, func() (bool, error) {
		ab, err = abclient.Get(ab.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get airflowbase: %s, %v", testname, err)
		}
		return ab.Status.Status == airflowv1alpha1.StatusReady, nil
	})
	Expect(err).NotTo(HaveOccurred(), "error waiting for airfloewbase to be ready: %s, %v", testname, err)

	return
}

func waitSts(client *kubernetes.Clientset, testname, rsrc, component string) {
	var stss *appsv1.StatefulSetList
	var err error
	stsClient := client.AppsV1().StatefulSets(namespace)
	selector := airflowv1alpha1.RsrcLabels(rsrc, testname, component)
	listop := metav1.ListOptions{LabelSelector: labels.Set(selector).AsSelector().String()}
	err = wait.PollImmediate(pollinterval, 2*time.Minute, func() (bool, error) {
		stss, err = stsClient.List(listop)
		if err != nil {
			return false, fmt.Errorf("failed to list sts: %s, %v", component, err)
		}
		return len(stss.Items) == 1, nil
	})
	Expect(err).NotTo(HaveOccurred(), "error waiting for list sts: %s, %v", component, err)

	sts := &stss.Items[0]
	err = wait.PollImmediate(pollinterval, 2*time.Minute, func() (bool, error) {
		sts, err = stsClient.Get(sts.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get sts: %s, %v", component, err)
		}
		return sts.Status.ReadyReplicas == *sts.Spec.Replicas && sts.Status.CurrentReplicas == *sts.Spec.Replicas, nil
	})
	Expect(err).NotTo(HaveOccurred(), "error waiting for sts to be ready: %s, %v", component, err)

	return
}

func rsrcName(name, component string) string {
	return name + "-" + component
}

func waitForSecret(client *kubernetes.Clientset, name, component string) {
	var err error
	secretClient := client.CoreV1().Secrets(namespace)

	err = wait.PollImmediate(pollinterval, 1*time.Minute, func() (bool, error) {
		_, err = secretClient.Get(rsrcName(name, component), metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	Expect(err).NotTo(HaveOccurred(), "error waiting for secret: %s, %v", component, err)

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

func clientConfig() *rest.Config {
	cc, err := clientcmd.BuildConfigFromFlags("", path.Join(os.Getenv("HOME"), ".kube/config"))
	if err != nil {
		log.Fatal("Unable to get client configuration", err)
	}
	return cc
}

func k8sClientSet() *kubernetes.Clientset {
	client, err := kubernetes.NewForConfig(clientConfig())
	if err != nil {
		log.Fatal("Unable to construct kubernetes client", err)
	}
	return client
}

func getDeleteOptions() *metav1.DeleteOptions {
	deletePropagationBackgroundPolicy := metav1.DeletePropagationBackground
	zero := int64(0)
	return &metav1.DeleteOptions{
		PropagationPolicy:  &deletePropagationBackgroundPolicy,
		GracePeriodSeconds: &zero,
	}
}

func getOperatorClientset() *typedairflow.AirflowV1alpha1Client {
	client, err := typedairflow.NewForConfig(clientConfig())
	if err != nil {
		log.Fatal("Unable to construct airflow operator client", err)
	}
	return client
}

var _ = BeforeSuite(func() {
	if true {
		By("Creating airflow controller statefulset")
		_, err := k8sClientSet().CoreV1().Namespaces().Create(operatorNamespace())
		Expect(err).NotTo(HaveOccurred(), "failed to create test namespace: %v", err)
		_, err = k8sClientSet().RbacV1().ClusterRoleBindings().Create(operatorCRB())
		Expect(err).NotTo(HaveOccurred(), "failed to create operator CRB: %v", err)
		_, err = k8sClientSet().AppsV1().StatefulSets(namespace).Create(operatorSts())
		Expect(err).NotTo(HaveOccurred(), "failed to create operator statefulset %q: %v", operatorsts, err)
	}
})

var _ = AfterSuite(func() {
	if true {
		By("Deleting airflow controller statefulset")
		err := k8sClientSet().AppsV1().StatefulSets(namespace).Delete(operatorsts, getDeleteOptions())
		Expect(err).NotTo(HaveOccurred(), "failed to delete operator statefulset %q: %v", operatorsts, err)
		err = k8sClientSet().RbacV1().ClusterRoleBindings().Delete(namespace+"crb", getDeleteOptions())
		Expect(err).NotTo(HaveOccurred(), "failed to delete operator CRB: %v", err)
		err = k8sClientSet().CoreV1().Namespaces().Delete(namespace, getDeleteOptions())
		Expect(err).NotTo(HaveOccurred(), "failed to delete test namespace: %v", err)
	}
})

var _ = Describe("AirflowBase and AirflowCluster Deployment should work", func() {
	var basem string
	k8sClient := k8sClientSet()
	airflowClient := getOperatorClientset()

	BeforeEach(func() {})
	AfterEach(func() {})

	// --------------- Tests -------------------------
	It("should create airflow-base using mysql and nfs components", func() { basem = testBaseComponentCreation(true, k8sClient, airflowClient) })

	// airflowBase() needs to inject sqlproxy config
	//It("should create airflow-base using sqlproxy and nfs components", func() { testBaseComponentCreation(false, k8sClient, airflowClient) })

	dags := &airflowv1alpha1.DagSpec{
		DagSubdir: "airflow/example_dags/",
		Git: &airflowv1alpha1.GitSpec{
			Repo: "https://github.com/apache/incubator-airflow/",
			Once: true,
		},
	}
	It("should create airflow-cluster using celery,git and using mysql base", func() {
		testClusterComponentCreation(basem, airflowv1alpha1.ExecutorCelery, k8sClient, airflowClient, dags)
	})

})

// Step 1: create AirflowBase object with storage and mysql/sqlproxy
// Step 2: wait for mysql/sqlproxy StatefulSet and nfs StatefulSet to become ready all pods have to be available
// Step 3: verify for mysql, the root password secret should be created, Mysql and nfs stateful sets should have 1 pods each
func testBaseComponentCreation(mysql bool, k8sClient *kubernetes.Clientset, airflowClient *typedairflow.AirflowV1alpha1Client) string {
	testname := "base"
	if !mysql {
		testname += "-sqlp"
	} else {
		testname += "-mysql"
	}
	_, err := airflowClient.AirflowBases(namespace).Create(airflowBase(testname, mysql))
	Expect(err).NotTo(HaveOccurred(), "failed to create AirflowBase %q: %v", testname, err)

	By(fmt.Sprintf("verifying AirflowBase %s components are created", testname))
	waitAirflowBase(k8sClient, airflowClient, testname, mysql)

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
func testClusterComponentCreation(base, executor string, k8sClient *kubernetes.Clientset, airflowClient *typedairflow.AirflowV1alpha1Client, dags *airflowv1alpha1.DagSpec) {
	testname := "cluster-cmg"
	_, err := airflowClient.AirflowClusters(namespace).Create(airflowCluster(testname, base, executor, dags))
	Expect(err).NotTo(HaveOccurred(), "failed to create AirflowCluster %q: %v", testname, err)

	By(fmt.Sprintf("verifying AirflowCluster %s components are created", testname))
	waitAirflowCluster(k8sClient, airflowClient, testname, executor)
}
