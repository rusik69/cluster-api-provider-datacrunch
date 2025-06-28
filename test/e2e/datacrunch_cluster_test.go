/*
Copyright 2024.

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

package e2e

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	infrastructurev1beta1 "github.com/rusik69/cluster-api-provider-datacrunch/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("DataCrunchCluster E2E", func() {
	var (
		mockAPI               *MockDataCrunchAPI
		testCluster           *infrastructurev1beta1.DataCrunchCluster
		testSecret            *corev1.Secret
		clusterName           string
		namespace             string
		extendedTimeout       = time.Minute * 5
		reconciliationTimeout = time.Second * 30
	)

	BeforeEach(func() {
		// Start mock API server
		mockAPI = NewMockDataCrunchAPI()

		// Set up test variables
		namespace = TestNamespace
		clusterName = "cluster-test-" + RandStringRunes(5)

		// Setup credentials
		SetupDataCrunchCredentials(mockAPI.URL())

		// Create secret for DataCrunch credentials
		testSecret = CreateSecret("datacrunch-credentials-"+RandStringRunes(5), namespace, mockAPI.URL())
		Expect(k8sClient.Create(ctx, testSecret)).To(Succeed())
	})

	AfterEach(func() {
		// Cleanup
		if testCluster != nil {
			Expect(k8sClient.Delete(ctx, testCluster)).To(Succeed())
			WaitForResourceDeletion(ctx, k8sClient, testCluster, extendedTimeout)
		}

		if testSecret != nil {
			Expect(k8sClient.Delete(ctx, testSecret)).To(Succeed())
			WaitForResourceDeletion(ctx, k8sClient, testSecret, timeout)
		}

		if mockAPI != nil {
			mockAPI.Close()
		}

		CleanupDataCrunchCredentials()
	})

	Context("Basic cluster lifecycle", func() {
		It("should create a DataCrunchCluster successfully", func() {
			By("Creating a DataCrunchCluster")
			testCluster = CreateDataCrunchCluster(clusterName, namespace)
			Expect(k8sClient.Create(ctx, testCluster)).To(Succeed())

			By("Verifying the cluster is created")
			Eventually(func() error {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      clusterName,
					Namespace: namespace,
				}, cluster)
			}, timeout, interval).Should(BeNil())

			By("Verifying cluster has expected spec values")
			cluster := &infrastructurev1beta1.DataCrunchCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      clusterName,
				Namespace: namespace,
			}, cluster)).To(Succeed())

			Expect(cluster.Spec.Region).To(Equal("FIN-01"))
			Expect(cluster.Spec.ControlPlaneEndpoint.Host).To(Equal(clusterName + ".datacrunch.local"))
			Expect(cluster.Spec.ControlPlaneEndpoint.Port).To(Equal(int32(6443)))
			Expect(cluster.Spec.Network.VPC.CidrBlock).To(Equal("10.0.0.0/16"))
		})

		It("should update cluster status during reconciliation", func() {
			By("Creating a DataCrunchCluster")
			testCluster = CreateDataCrunchCluster(clusterName, namespace)
			Expect(k8sClient.Create(ctx, testCluster)).To(Succeed())

			By("Waiting for cluster status to be updated")
			Eventually(func() bool {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      clusterName,
					Namespace: namespace,
				}, cluster)
				if err != nil {
					return false
				}
				return len(cluster.Status.Conditions) > 0
			}, reconciliationTimeout, interval).Should(BeTrue())

			By("Verifying cluster has infrastructure ready condition")
			cluster := &infrastructurev1beta1.DataCrunchCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      clusterName,
				Namespace: namespace,
			}, cluster)).To(Succeed())

			// Check for InfrastructureReady condition
			Eventually(func() bool {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      clusterName,
					Namespace: namespace,
				}, cluster)
				if err != nil {
					return false
				}
				return HasCondition(cluster.Status.Conditions, clusterv1.ReadyCondition)
			}, reconciliationTimeout, interval).Should(BeTrue())
		})

		It("should handle cluster deletion properly", func() {
			By("Creating a DataCrunchCluster")
			testCluster = CreateDataCrunchCluster(clusterName, namespace)
			Expect(k8sClient.Create(ctx, testCluster)).To(Succeed())

			By("Waiting for cluster to be created")
			Eventually(func() error {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      clusterName,
					Namespace: namespace,
				}, cluster)
			}, timeout, interval).Should(BeNil())

			By("Deleting the cluster")
			Expect(k8sClient.Delete(ctx, testCluster)).To(Succeed())

			By("Verifying cluster is deleted")
			WaitForResourceDeletion(ctx, k8sClient, testCluster, extendedTimeout)

			// Set to nil so AfterEach doesn't try to delete again
			testCluster = nil
		})
	})

	Context("Cluster specifications", func() {
		It("should create cluster with custom network configuration", func() {
			By("Creating a cluster with custom network settings")
			testCluster = &infrastructurev1beta1.DataCrunchCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: namespace,
				},
				Spec: infrastructurev1beta1.DataCrunchClusterSpec{
					Region: "FIN-01",
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: clusterName + ".custom.local",
						Port: 8443,
					},
					ControlPlaneLoadBalancer: &infrastructurev1beta1.DataCrunchLoadBalancerSpec{
						Type: "internal",
					},
					Network: &infrastructurev1beta1.DataCrunchNetworkSpec{
						VPC: &infrastructurev1beta1.DataCrunchVPCSpec{
							CidrBlock: "172.16.0.0/16",
						},
						Subnets: []infrastructurev1beta1.DataCrunchSubnetSpec{
							{
								CidrBlock:        "172.16.1.0/24",
								AvailabilityZone: "FIN-01a",
							},
							{
								CidrBlock:        "172.16.2.0/24",
								AvailabilityZone: "FIN-01b",
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, testCluster)).To(Succeed())

			By("Verifying custom configuration is preserved")
			cluster := &infrastructurev1beta1.DataCrunchCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      clusterName,
				Namespace: namespace,
			}, cluster)).To(Succeed())

			Expect(cluster.Spec.ControlPlaneEndpoint.Host).To(Equal(clusterName + ".custom.local"))
			Expect(cluster.Spec.ControlPlaneEndpoint.Port).To(Equal(int32(8443)))
			Expect(cluster.Spec.Network.VPC.CidrBlock).To(Equal("172.16.0.0/16"))
			Expect(len(cluster.Spec.Network.Subnets)).To(Equal(2))
			Expect(cluster.Spec.ControlPlaneLoadBalancer.Type).To(Equal("internal"))
		})

		It("should create cluster without load balancer", func() {
			By("Creating a cluster without load balancer")
			testCluster = &infrastructurev1beta1.DataCrunchCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterName,
					Namespace: namespace,
				},
				Spec: infrastructurev1beta1.DataCrunchClusterSpec{
					Region: "FIN-01",
					ControlPlaneEndpoint: clusterv1.APIEndpoint{
						Host: clusterName + ".datacrunch.local",
						Port: 6443,
					},
					// No load balancer specified
					Network: &infrastructurev1beta1.DataCrunchNetworkSpec{
						VPC: &infrastructurev1beta1.DataCrunchVPCSpec{
							CidrBlock: "10.0.0.0/16",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, testCluster)).To(Succeed())

			By("Verifying cluster is created without load balancer")
			cluster := &infrastructurev1beta1.DataCrunchCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      clusterName,
				Namespace: namespace,
			}, cluster)).To(Succeed())

			Expect(cluster.Spec.ControlPlaneLoadBalancer).To(BeNil())
		})
	})

	Context("Cluster status and conditions", func() {
		BeforeEach(func() {
			testCluster = CreateDataCrunchCluster(clusterName, namespace)
			Expect(k8sClient.Create(ctx, testCluster)).To(Succeed())
		})

		It("should report cluster readiness status", func() {
			By("Waiting for cluster to become ready")
			Eventually(func() bool {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      clusterName,
					Namespace: namespace,
				}, cluster)
				if err != nil {
					return false
				}

				// Check if Ready condition exists and is True
				readyCondition := GetCondition(cluster.Status.Conditions, clusterv1.ReadyCondition)
				return readyCondition != nil && readyCondition.Status == corev1.ConditionTrue
			}, extendedTimeout, interval).Should(BeTrue())

			By("Verifying cluster ready status is true")
			cluster := &infrastructurev1beta1.DataCrunchCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      clusterName,
				Namespace: namespace,
			}, cluster)).To(Succeed())

			Eventually(func() bool {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      clusterName,
					Namespace: namespace,
				}, cluster)
				return err == nil && cluster.Status.Ready
			}, extendedTimeout, interval).Should(BeTrue())
		})

		It("should have infrastructure ready condition", func() {
			By("Waiting for infrastructure ready condition")
			WaitForClusterCondition(ctx, k8sClient, clusterName, namespace,
				clusterv1.ReadyCondition, corev1.ConditionTrue, extendedTimeout)

			By("Verifying condition details")
			cluster := &infrastructurev1beta1.DataCrunchCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      clusterName,
				Namespace: namespace,
			}, cluster)).To(Succeed())

			readyCondition := GetCondition(cluster.Status.Conditions, clusterv1.ReadyCondition)
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(corev1.ConditionTrue))
			Expect(readyCondition.Reason).NotTo(BeEmpty())
		})
	})

	Context("Error handling", func() {
		It("should handle invalid region gracefully", func() {
			By("Creating a cluster with invalid region")
			testCluster = CreateDataCrunchCluster(clusterName, namespace)
			testCluster.Spec.Region = "INVALID-REGION"
			Expect(k8sClient.Create(ctx, testCluster)).To(Succeed())

			By("Waiting for error condition to be set")
			Eventually(func() bool {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      clusterName,
					Namespace: namespace,
				}, cluster)
				if err != nil {
					return false
				}

				// Look for any condition with False status indicating an error
				for _, condition := range cluster.Status.Conditions {
					if condition.Status == corev1.ConditionFalse {
						return true
					}
				}
				return false
			}, reconciliationTimeout, interval).Should(BeTrue())
		})

		It("should handle missing credentials gracefully", func() {
			By("Deleting the credentials secret")
			Expect(k8sClient.Delete(ctx, testSecret)).To(Succeed())
			WaitForResourceDeletion(ctx, k8sClient, testSecret, timeout)
			testSecret = nil // Prevent double deletion in AfterEach

			By("Creating a cluster without credentials")
			testCluster = CreateDataCrunchCluster(clusterName, namespace)
			Expect(k8sClient.Create(ctx, testCluster)).To(Succeed())

			By("Verifying cluster shows error condition due to missing credentials")
			Eventually(func() bool {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      clusterName,
					Namespace: namespace,
				}, cluster)
				if err != nil {
					return false
				}

				// Should not be ready due to missing credentials
				return !cluster.Status.Ready
			}, reconciliationTimeout, interval).Should(BeTrue())
		})
	})
})

// RandStringRunes generates a random string of specified length
func RandStringRunes(n int) string {
	const letterRunes = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterRunes[i%len(letterRunes)]
	}
	return string(b)
}
