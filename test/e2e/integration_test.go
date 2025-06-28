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
	"k8s.io/apimachinery/pkg/types"

	infrastructurev1beta1 "github.com/rusik69/cluster-api-provider-datacrunch/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var _ = Describe("DataCrunch Provider Integration E2E", func() {
	var (
		mockAPI         *MockDataCrunchAPI
		testCluster     *infrastructurev1beta1.DataCrunchCluster
		testMachine     *infrastructurev1beta1.DataCrunchMachine
		testSecret      *corev1.Secret
		clusterName     string
		machineName     string
		namespace       string
		extendedTimeout = time.Minute * 10
	)

	BeforeEach(func() {
		// Start mock API server
		mockAPI = NewMockDataCrunchAPI()

		// Set up test variables
		namespace = TestNamespace
		clusterName = "integration-cluster-" + RandStringRunes(5)
		machineName = "integration-machine-" + RandStringRunes(5)

		// Setup credentials
		SetupDataCrunchCredentials(mockAPI.URL())

		// Create secret for DataCrunch credentials
		testSecret = CreateSecret("datacrunch-credentials-"+RandStringRunes(5), namespace, mockAPI.URL())
		Expect(k8sClient.Create(ctx, testSecret)).To(Succeed())
	})

	AfterEach(func() {
		// Cleanup in reverse order (machine first, then cluster)
		if testMachine != nil {
			Expect(k8sClient.Delete(ctx, testMachine)).To(Succeed())
			WaitForResourceDeletion(ctx, k8sClient, testMachine, extendedTimeout)
		}

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

	Context("Full cluster and machine lifecycle", func() {
		It("should create and manage a complete DataCrunch infrastructure", func() {
			By("Creating a DataCrunchCluster")
			testCluster = CreateDataCrunchCluster(clusterName, namespace)
			Expect(k8sClient.Create(ctx, testCluster)).To(Succeed())

			By("Waiting for cluster to be ready")
			Eventually(func() bool {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      clusterName,
					Namespace: namespace,
				}, cluster)
				if err != nil {
					return false
				}
				return cluster.Status.Ready
			}, extendedTimeout, interval).Should(BeTrue())

			By("Creating a DataCrunchMachine")
			testMachine = CreateDataCrunchMachine(machineName, namespace)
			// Link machine to cluster via labels (simulation of CAPI integration)
			testMachine.Labels = map[string]string{
				clusterv1.ClusterNameLabel: clusterName,
			}
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Waiting for machine to be ready")
			Eventually(func() bool {
				machine := &infrastructurev1beta1.DataCrunchMachine{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      machineName,
					Namespace: namespace,
				}, machine)
				if err != nil {
					return false
				}
				return machine.Status.Ready
			}, extendedTimeout, interval).Should(BeTrue())

			By("Verifying machine has running instance")
			machine := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      machineName,
				Namespace: namespace,
			}, machine)).To(Succeed())

			Expect(machine.Status.InstanceState).NotTo(BeNil())
			Expect(*machine.Status.InstanceState).To(Equal(infrastructurev1beta1.InstanceStateRunning))
			Expect(len(machine.Status.Addresses)).To(BeNumerically(">", 0))

			By("Verifying both resources have appropriate conditions")
			cluster := &infrastructurev1beta1.DataCrunchCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      clusterName,
				Namespace: namespace,
			}, cluster)).To(Succeed())

			Expect(HasCondition(cluster.Status.Conditions, clusterv1.ReadyCondition)).To(BeTrue())
			Expect(HasCondition(machine.Status.Conditions, clusterv1.ReadyCondition)).To(BeTrue())
		})

		It("should handle machine deletion before cluster deletion", func() {
			By("Creating cluster and machine")
			testCluster = CreateDataCrunchCluster(clusterName, namespace)
			Expect(k8sClient.Create(ctx, testCluster)).To(Succeed())

			testMachine = CreateDataCrunchMachine(machineName, namespace)
			testMachine.Labels = map[string]string{
				clusterv1.ClusterNameLabel: clusterName,
			}
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Waiting for both to be ready")
			Eventually(func() bool {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				machine := &infrastructurev1beta1.DataCrunchMachine{}

				clusterErr := k8sClient.Get(ctx, types.NamespacedName{
					Name: clusterName, Namespace: namespace,
				}, cluster)
				machineErr := k8sClient.Get(ctx, types.NamespacedName{
					Name: machineName, Namespace: namespace,
				}, machine)

				return clusterErr == nil && machineErr == nil &&
					cluster.Status.Ready && machine.Status.Ready
			}, extendedTimeout, interval).Should(BeTrue())

			By("Deleting the machine first")
			Expect(k8sClient.Delete(ctx, testMachine)).To(Succeed())
			WaitForResourceDeletion(ctx, k8sClient, testMachine, extendedTimeout)
			testMachine = nil

			By("Verifying cluster is still ready")
			cluster := &infrastructurev1beta1.DataCrunchCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      clusterName,
				Namespace: namespace,
			}, cluster)).To(Succeed())
			Expect(cluster.Status.Ready).To(BeTrue())

			By("Deleting the cluster")
			Expect(k8sClient.Delete(ctx, testCluster)).To(Succeed())
			WaitForResourceDeletion(ctx, k8sClient, testCluster, extendedTimeout)
			testCluster = nil
		})
	})

	Context("Multi-machine cluster scenarios", func() {
		var (
			controlPlaneMachine *infrastructurev1beta1.DataCrunchMachine
			workerMachine1      *infrastructurev1beta1.DataCrunchMachine
			workerMachine2      *infrastructurev1beta1.DataCrunchMachine
		)

		AfterEach(func() {
			// Cleanup machines
			machines := []*infrastructurev1beta1.DataCrunchMachine{
				workerMachine2, workerMachine1, controlPlaneMachine,
			}

			for _, machine := range machines {
				if machine != nil {
					Expect(k8sClient.Delete(ctx, machine)).To(Succeed())
					WaitForResourceDeletion(ctx, k8sClient, machine, extendedTimeout)
				}
			}
		})

		It("should create cluster with multiple machines", func() {
			By("Creating a DataCrunchCluster")
			testCluster = CreateDataCrunchCluster(clusterName, namespace)
			Expect(k8sClient.Create(ctx, testCluster)).To(Succeed())

			By("Waiting for cluster to be ready")
			Eventually(func() bool {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      clusterName,
					Namespace: namespace,
				}, cluster)
				return err == nil && cluster.Status.Ready
			}, extendedTimeout, interval).Should(BeTrue())

			By("Creating control plane machine")
			controlPlaneMachine = CreateDataCrunchMachine("cp-"+machineName, namespace)
			controlPlaneMachine.Labels = map[string]string{
				clusterv1.ClusterNameLabel:         clusterName,
				clusterv1.MachineControlPlaneLabel: "true",
			}
			controlPlaneMachine.Spec.InstanceType = "4vcpu-16gb" // Smaller for control plane
			Expect(k8sClient.Create(ctx, controlPlaneMachine)).To(Succeed())

			By("Creating worker machine 1")
			workerMachine1 = CreateDataCrunchMachine("w1-"+machineName, namespace)
			workerMachine1.Labels = map[string]string{
				clusterv1.ClusterNameLabel: clusterName,
			}
			workerMachine1.Spec.InstanceType = "1xH100" // GPU for ML workloads
			Expect(k8sClient.Create(ctx, workerMachine1)).To(Succeed())

			By("Creating worker machine 2")
			workerMachine2 = CreateDataCrunchMachine("w2-"+machineName, namespace)
			workerMachine2.Labels = map[string]string{
				clusterv1.ClusterNameLabel: clusterName,
			}
			workerMachine2.Spec.InstanceType = "1xA100" // Different GPU type
			Expect(k8sClient.Create(ctx, workerMachine2)).To(Succeed())

			By("Waiting for all machines to be ready")
			machines := []*infrastructurev1beta1.DataCrunchMachine{
				controlPlaneMachine, workerMachine1, workerMachine2,
			}

			for _, machine := range machines {
				Eventually(func() bool {
					m := &infrastructurev1beta1.DataCrunchMachine{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name:      machine.Name,
						Namespace: namespace,
					}, m)
					return err == nil && m.Status.Ready
				}, extendedTimeout, interval).Should(BeTrue())
			}

			By("Verifying all machines have correct instance types")
			cp := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: controlPlaneMachine.Name, Namespace: namespace,
			}, cp)).To(Succeed())
			Expect(cp.Spec.InstanceType).To(Equal("4vcpu-16gb"))

			w1 := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: workerMachine1.Name, Namespace: namespace,
			}, w1)).To(Succeed())
			Expect(w1.Spec.InstanceType).To(Equal("1xH100"))

			w2 := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: workerMachine2.Name, Namespace: namespace,
			}, w2)).To(Succeed())
			Expect(w2.Spec.InstanceType).To(Equal("1xA100"))
		})
	})

	Context("Error recovery scenarios", func() {
		It("should recover from temporary API failures", func() {
			By("Creating cluster and machine")
			testCluster = CreateDataCrunchCluster(clusterName, namespace)
			Expect(k8sClient.Create(ctx, testCluster)).To(Succeed())

			testMachine = CreateDataCrunchMachine(machineName, namespace)
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Waiting for initial readiness")
			Eventually(func() bool {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				machine := &infrastructurev1beta1.DataCrunchMachine{}

				clusterErr := k8sClient.Get(ctx, types.NamespacedName{
					Name: clusterName, Namespace: namespace,
				}, cluster)
				machineErr := k8sClient.Get(ctx, types.NamespacedName{
					Name: machineName, Namespace: namespace,
				}, machine)

				return clusterErr == nil && machineErr == nil &&
					cluster.Status.Ready && machine.Status.Ready
			}, extendedTimeout, interval).Should(BeTrue())

			By("Simulating API recovery by restarting mock server")
			oldURL := mockAPI.URL()
			mockAPI.Close()

			// Start new mock server
			mockAPI = NewMockDataCrunchAPI()
			SetupDataCrunchCredentials(mockAPI.URL())

			// Update secret with new URL
			secret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: testSecret.Name, Namespace: namespace,
			}, secret)).To(Succeed())
			secret.Data["apiURL"] = []byte(mockAPI.URL())
			Expect(k8sClient.Update(ctx, secret)).To(Succeed())

			By("Verifying resources maintain their state after API recovery")
			Consistently(func() bool {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				machine := &infrastructurev1beta1.DataCrunchMachine{}

				clusterErr := k8sClient.Get(ctx, types.NamespacedName{
					Name: clusterName, Namespace: namespace,
				}, cluster)
				machineErr := k8sClient.Get(ctx, types.NamespacedName{
					Name: machineName, Namespace: namespace,
				}, machine)

				if clusterErr != nil || machineErr != nil {
					return false
				}

				// Resources should remain ready or become ready again
				return cluster.Status.Ready && machine.Status.Ready
			}, time.Second*10, interval).Should(BeTrue())

			GinkgoWriter.Printf("API recovered from %s to %s\n", oldURL, mockAPI.URL())
		})

		It("should handle credentials rotation", func() {
			By("Creating cluster and machine with initial credentials")
			testCluster = CreateDataCrunchCluster(clusterName, namespace)
			Expect(k8sClient.Create(ctx, testCluster)).To(Succeed())

			testMachine = CreateDataCrunchMachine(machineName, namespace)
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Waiting for resources to be ready")
			Eventually(func() bool {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				machine := &infrastructurev1beta1.DataCrunchMachine{}

				clusterErr := k8sClient.Get(ctx, types.NamespacedName{
					Name: clusterName, Namespace: namespace,
				}, cluster)
				machineErr := k8sClient.Get(ctx, types.NamespacedName{
					Name: machineName, Namespace: namespace,
				}, machine)

				return clusterErr == nil && machineErr == nil &&
					cluster.Status.Ready && machine.Status.Ready
			}, extendedTimeout, interval).Should(BeTrue())

			By("Rotating credentials in the secret")
			secret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: testSecret.Name, Namespace: namespace,
			}, secret)).To(Succeed())

			// Update credentials
			secret.Data["clientID"] = []byte("new-client-id")
			secret.Data["clientSecret"] = []byte("new-client-secret")
			Expect(k8sClient.Update(ctx, secret)).To(Succeed())

			// Update environment variables
			SetupDataCrunchCredentials(mockAPI.URL())

			By("Verifying resources continue to work with new credentials")
			Consistently(func() bool {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				machine := &infrastructurev1beta1.DataCrunchMachine{}

				clusterErr := k8sClient.Get(ctx, types.NamespacedName{
					Name: clusterName, Namespace: namespace,
				}, cluster)
				machineErr := k8sClient.Get(ctx, types.NamespacedName{
					Name: machineName, Namespace: namespace,
				}, machine)

				return clusterErr == nil && machineErr == nil &&
					cluster.Status.Ready && machine.Status.Ready
			}, time.Second*15, interval).Should(BeTrue())
		})
	})

	Context("Resource scaling scenarios", func() {
		It("should handle machine scaling operations", func() {
			By("Creating a cluster")
			testCluster = CreateDataCrunchCluster(clusterName, namespace)
			Expect(k8sClient.Create(ctx, testCluster)).To(Succeed())

			By("Waiting for cluster to be ready")
			Eventually(func() bool {
				cluster := &infrastructurev1beta1.DataCrunchCluster{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: clusterName, Namespace: namespace,
				}, cluster)
				return err == nil && cluster.Status.Ready
			}, extendedTimeout, interval).Should(BeTrue())

			By("Creating initial machine")
			testMachine = CreateDataCrunchMachine(machineName, namespace)
			testMachine.Labels = map[string]string{
				clusterv1.ClusterNameLabel: clusterName,
			}
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Waiting for machine to be ready")
			Eventually(func() bool {
				machine := &infrastructurev1beta1.DataCrunchMachine{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name: machineName, Namespace: namespace,
				}, machine)
				return err == nil && machine.Status.Ready
			}, extendedTimeout, interval).Should(BeTrue())

			By("Scaling up by creating additional machines")
			additionalMachines := make([]*infrastructurev1beta1.DataCrunchMachine, 2)
			for i := 0; i < 2; i++ {
				additionalMachine := CreateDataCrunchMachine(
					machineName+"-scale-"+string(rune('a'+i)), namespace)
				additionalMachine.Labels = map[string]string{
					clusterv1.ClusterNameLabel: clusterName,
				}
				additionalMachine.Spec.InstanceType = "4vcpu-16gb" // Smaller instances
				Expect(k8sClient.Create(ctx, additionalMachine)).To(Succeed())
				additionalMachines[i] = additionalMachine
			}

			By("Waiting for all additional machines to be ready")
			for i, machine := range additionalMachines {
				Eventually(func() bool {
					m := &infrastructurev1beta1.DataCrunchMachine{}
					err := k8sClient.Get(ctx, types.NamespacedName{
						Name: machine.Name, Namespace: namespace,
					}, m)
					return err == nil && m.Status.Ready
				}, extendedTimeout, interval).Should(BeTrue(), "Machine %d should be ready", i)
			}

			By("Scaling down by removing additional machines")
			for _, machine := range additionalMachines {
				Expect(k8sClient.Delete(ctx, machine)).To(Succeed())
				WaitForResourceDeletion(ctx, k8sClient, machine, extendedTimeout)
			}

			By("Verifying original machine and cluster remain ready")
			cluster := &infrastructurev1beta1.DataCrunchCluster{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: clusterName, Namespace: namespace,
			}, cluster)).To(Succeed())
			Expect(cluster.Status.Ready).To(BeTrue())

			machine := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name: machineName, Namespace: namespace,
			}, machine)).To(Succeed())
			Expect(machine.Status.Ready).To(BeTrue())
		})
	})
})
