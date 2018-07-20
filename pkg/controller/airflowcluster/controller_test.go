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

package airflowcluster_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "k8s.io/airflow-operator/pkg/apis/airflow/v1alpha1"
	. "k8s.io/airflow-operator/pkg/client/clientset/versioned/typed/airflow/v1alpha1"
)

// EDIT THIS FILE!
// Created by "kubebuilder create resource" for you to implement controller logic tests

var _ = Describe("AirflowCluster controller", func() {
	var instance AirflowCluster
	var expectedKey types.ReconcileKey
	var client AirflowClusterInterface

	BeforeEach(func() {
		instance = AirflowCluster{}
		instance.Name = "instance-1"
		instance.Spec.AirflowBaseRef = &corev1.LocalObjectReference{Name: "base-1"}
		instance.Spec.Scheduler = &SchedulerSpec{Image: ""}
		expectedKey = types.ReconcileKey{
			Namespace: "default",
			Name:      "instance-1",
		}
	})

	AfterEach(func() {
		client.Delete(instance.Name, &metav1.DeleteOptions{})
	})

	Describe("when creating a new object", func() {
		It("invoke the reconcile method", func() {
			after := make(chan struct{})
			ctrl.AfterReconcile = func(key types.ReconcileKey, err error) {
				defer func() {
					// Recover in case the key is reconciled multiple times
					defer func() { recover() }()
					close(after)
				}()
				defer GinkgoRecover()
				Expect(key).To(Equal(expectedKey))
				Expect(err).ToNot(HaveOccurred())
			}

			// Create the instance
			client = cs.AirflowV1alpha1().AirflowClusters("default")
			_, err := client.Create(&instance)
			Expect(err).ShouldNot(HaveOccurred())

			// Wait for reconcile to happen
			Eventually(after, "10s", "100ms").Should(BeClosed())

			// INSERT YOUR CODE HERE - test conditions post reconcile
		})
	})
})
