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

var _ = Describe("DataCrunchMachine E2E", func() {
	var (
		mockAPI               *MockDataCrunchAPI
		testMachine           *infrastructurev1beta1.DataCrunchMachine
		testSecret            *corev1.Secret
		machineName           string
		namespace             string
		extendedTimeout       = time.Minute * 5
		reconciliationTimeout = time.Second * 30
	)

	BeforeEach(func() {
		// Start mock API server
		mockAPI = NewMockDataCrunchAPI()

		// Set up test variables
		namespace = TestNamespace
		machineName = "machine-test-" + RandStringRunes(5)

		// Setup credentials
		SetupDataCrunchCredentials(mockAPI.URL())

		// Create secret for DataCrunch credentials
		testSecret = CreateSecret("datacrunch-credentials-"+RandStringRunes(5), namespace, mockAPI.URL())
		Expect(k8sClient.Create(ctx, testSecret)).To(Succeed())
	})

	AfterEach(func() {
		// Cleanup
		if testMachine != nil {
			Expect(k8sClient.Delete(ctx, testMachine)).To(Succeed())
			WaitForResourceDeletion(ctx, k8sClient, testMachine, extendedTimeout)
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

	Context("Basic machine lifecycle", func() {
		It("should create a DataCrunchMachine successfully", func() {
			By("Creating a DataCrunchMachine")
			testMachine = CreateDataCrunchMachine(machineName, namespace)
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Verifying the machine is created")
			Eventually(func() error {
				machine := &infrastructurev1beta1.DataCrunchMachine{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      machineName,
					Namespace: namespace,
				}, machine)
			}, timeout, interval).Should(BeNil())

			By("Verifying machine has expected spec values")
			machine := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      machineName,
				Namespace: namespace,
			}, machine)).To(Succeed())

			Expect(machine.Spec.InstanceType).To(Equal("1xH100"))
			Expect(machine.Spec.Image).To(Equal("ubuntu-22.04-cuda-12.1"))
			Expect(machine.Spec.SSHKeyName).To(Equal("test-key"))
			Expect(*machine.Spec.PublicIP).To(BeTrue())
			Expect(machine.Spec.RootVolume.Size).To(Equal(100))
			Expect(machine.Spec.RootVolume.Type).To(Equal("fast-ssd"))
		})

		It("should update machine status during reconciliation", func() {
			By("Creating a DataCrunchMachine")
			testMachine = CreateDataCrunchMachine(machineName, namespace)
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Waiting for machine status to be updated")
			Eventually(func() bool {
				machine := &infrastructurev1beta1.DataCrunchMachine{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      machineName,
					Namespace: namespace,
				}, machine)
				if err != nil {
					return false
				}
				return len(machine.Status.Conditions) > 0
			}, reconciliationTimeout, interval).Should(BeTrue())

			By("Verifying machine has ready condition")
			Eventually(func() bool {
				machine := &infrastructurev1beta1.DataCrunchMachine{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      machineName,
					Namespace: namespace,
				}, machine)
				if err != nil {
					return false
				}
				return HasCondition(machine.Status.Conditions, clusterv1.ReadyCondition)
			}, reconciliationTimeout, interval).Should(BeTrue())
		})

		It("should handle machine deletion properly", func() {
			By("Creating a DataCrunchMachine")
			testMachine = CreateDataCrunchMachine(machineName, namespace)
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Waiting for machine to be created")
			Eventually(func() error {
				machine := &infrastructurev1beta1.DataCrunchMachine{}
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      machineName,
					Namespace: namespace,
				}, machine)
			}, timeout, interval).Should(BeNil())

			By("Deleting the machine")
			Expect(k8sClient.Delete(ctx, testMachine)).To(Succeed())

			By("Verifying machine is deleted")
			WaitForResourceDeletion(ctx, k8sClient, testMachine, extendedTimeout)

			// Set to nil so AfterEach doesn't try to delete again
			testMachine = nil
		})
	})

	Context("Machine instance types and configurations", func() {
		It("should create machine with H100 GPU instance", func() {
			By("Creating a machine with H100 GPU")
			testMachine = CreateDataCrunchMachine(machineName, namespace)
			testMachine.Spec.InstanceType = "1xH100"
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Verifying instance type is set correctly")
			machine := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      machineName,
				Namespace: namespace,
			}, machine)).To(Succeed())

			Expect(machine.Spec.InstanceType).To(Equal("1xH100"))
		})

		It("should create machine with multiple GPUs", func() {
			By("Creating a machine with multiple H100 GPUs")
			testMachine = CreateDataCrunchMachine(machineName, namespace)
			testMachine.Spec.InstanceType = "8xH100"
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Verifying multi-GPU instance type")
			machine := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      machineName,
				Namespace: namespace,
			}, machine)).To(Succeed())

			Expect(machine.Spec.InstanceType).To(Equal("8xH100"))
		})

		It("should create CPU-only machine", func() {
			By("Creating a CPU-only machine")
			testMachine = CreateDataCrunchMachine(machineName, namespace)
			testMachine.Spec.InstanceType = "4vcpu-16gb"
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Verifying CPU instance type")
			machine := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      machineName,
				Namespace: namespace,
			}, machine)).To(Succeed())

			Expect(machine.Spec.InstanceType).To(Equal("4vcpu-16gb"))
		})

		It("should create machine with custom root volume configuration", func() {
			By("Creating a machine with custom root volume")
			testMachine = CreateDataCrunchMachine(machineName, namespace)
			testMachine.Spec.RootVolume = &infrastructurev1beta1.Volume{
				Size:      500,
				Type:      "nvme-ssd",
				Encrypted: func() *bool { b := true; return &b }(),
			}
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Verifying root volume configuration")
			machine := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      machineName,
				Namespace: namespace,
			}, machine)).To(Succeed())

			Expect(machine.Spec.RootVolume).NotTo(BeNil())
			Expect(machine.Spec.RootVolume.Size).To(Equal(int64(500)))
			Expect(machine.Spec.RootVolume.Type).To(Equal("nvme-ssd"))
			Expect(*machine.Spec.RootVolume.Encrypted).To(BeTrue())
		})
	})

	Context("Machine status and instance states", func() {
		BeforeEach(func() {
			testMachine = CreateDataCrunchMachine(machineName, namespace)
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())
		})

		It("should transition machine to running state", func() {
			By("Waiting for machine to reach running state")
			Eventually(func() bool {
				machine := &infrastructurev1beta1.DataCrunchMachine{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      machineName,
					Namespace: namespace,
				}, machine)
				if err != nil {
					return false
				}

				return machine.Status.InstanceState != nil &&
					*machine.Status.InstanceState == infrastructurev1beta1.InstanceStateRunning
			}, extendedTimeout, interval).Should(BeTrue())

			By("Verifying machine is ready")
			Eventually(func() bool {
				machine := &infrastructurev1beta1.DataCrunchMachine{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      machineName,
					Namespace: namespace,
				}, machine)
				return err == nil && machine.Status.Ready
			}, extendedTimeout, interval).Should(BeTrue())
		})

		It("should set instance addresses", func() {
			By("Waiting for instance addresses to be set")
			Eventually(func() bool {
				machine := &infrastructurev1beta1.DataCrunchMachine{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      machineName,
					Namespace: namespace,
				}, machine)
				if err != nil {
					return false
				}

				return len(machine.Status.Addresses) > 0
			}, extendedTimeout, interval).Should(BeTrue())

			By("Verifying address types and values")
			machine := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      machineName,
				Namespace: namespace,
			}, machine)).To(Succeed())

			hasPublicIP := false
			hasPrivateIP := false
			for _, addr := range machine.Status.Addresses {
				if addr.Type == clusterv1.MachineExternalIP {
					hasPublicIP = true
					Expect(addr.Address).NotTo(BeEmpty())
				}
				if addr.Type == clusterv1.MachineInternalIP {
					hasPrivateIP = true
					Expect(addr.Address).NotTo(BeEmpty())
				}
			}

			Expect(hasPublicIP).To(BeTrue())
			Expect(hasPrivateIP).To(BeTrue())
		})

		It("should have infrastructure ready condition", func() {
			By("Waiting for infrastructure ready condition")
			WaitForMachineCondition(ctx, k8sClient, machineName, namespace,
				clusterv1.ReadyCondition, corev1.ConditionTrue, extendedTimeout)

			By("Verifying condition details")
			machine := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      machineName,
				Namespace: namespace,
			}, machine)).To(Succeed())

			readyCondition := GetCondition(machine.Status.Conditions, clusterv1.ReadyCondition)
			Expect(readyCondition).NotTo(BeNil())
			Expect(readyCondition.Status).To(Equal(corev1.ConditionTrue))
			Expect(readyCondition.Reason).NotTo(BeEmpty())
		})
	})

	Context("Machine with spot instances", func() {
		It("should create spot instance successfully", func() {
			By("Creating a machine with spot instance configuration")
			testMachine = CreateDataCrunchMachine(machineName, namespace)
			testMachine.Spec.Spot = &infrastructurev1beta1.SpotMachineOptions{
				MaxPrice: func() *string { s := "0.50"; return &s }(),
			}
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Verifying spot instance configuration")
			machine := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      machineName,
				Namespace: namespace,
			}, machine)).To(Succeed())

			Expect(machine.Spec.Spot).NotTo(BeNil())
			Expect(*machine.Spec.Spot.MaxPrice).To(Equal("0.50"))
		})
	})

	Context("Machine failure handling", func() {
		BeforeEach(func() {
			testMachine = CreateDataCrunchMachine(machineName, namespace)
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())
		})

		It("should handle instance creation failure", func() {
			By("Updating machine with invalid instance type to trigger failure")
			machine := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      machineName,
				Namespace: namespace,
			}, machine)).To(Succeed())

			machine.Spec.InstanceType = "invalid-instance-type"
			Expect(k8sClient.Update(ctx, machine)).To(Succeed())

			By("Waiting for failure condition to be set")
			Eventually(func() bool {
				machine := &infrastructurev1beta1.DataCrunchMachine{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      machineName,
					Namespace: namespace,
				}, machine)
				if err != nil {
					return false
				}

				// Look for failure condition or machine not ready
				return !machine.Status.Ready ||
					machine.Status.FailureReason != nil ||
					machine.Status.FailureMessage != nil
			}, reconciliationTimeout, interval).Should(BeTrue())
		})

		It("should set failure reason and message on API errors", func() {
			By("Creating machine with configuration that will cause API error")
			testMachine.Spec.Image = "non-existent-image"
			Expect(k8sClient.Update(ctx, testMachine)).To(Succeed())

			By("Waiting for failure information to be set")
			Eventually(func() bool {
				machine := &infrastructurev1beta1.DataCrunchMachine{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      machineName,
					Namespace: namespace,
				}, machine)
				if err != nil {
					return false
				}

				return machine.Status.FailureReason != nil || machine.Status.FailureMessage != nil
			}, reconciliationTimeout, interval).Should(BeTrue())

			By("Verifying failure details are populated")
			machine := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      machineName,
				Namespace: namespace,
			}, machine)).To(Succeed())

			// At least one failure field should be set
			Expect(machine.Status.FailureReason != nil || machine.Status.FailureMessage != nil).To(BeTrue())
		})
	})

	Context("Machine metadata and tagging", func() {
		It("should create machine with custom tags and metadata", func() {
			By("Creating a machine with custom tags and metadata")
			testMachine = CreateDataCrunchMachine(machineName, namespace)
			testMachine.Spec.AdditionalTags = map[string]string{
				"project":     "ml-training",
				"environment": "production",
				"team":        "data-science",
			}
			testMachine.Spec.AdditionalMetadata = map[string]string{
				"gpu-workload": "training",
				"framework":    "pytorch",
				"dataset":      "imagenet",
			}
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Verifying tags and metadata are preserved")
			machine := &infrastructurev1beta1.DataCrunchMachine{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      machineName,
				Namespace: namespace,
			}, machine)).To(Succeed())

			Expect(machine.Spec.AdditionalTags["project"]).To(Equal("ml-training"))
			Expect(machine.Spec.AdditionalTags["environment"]).To(Equal("production"))
			Expect(machine.Spec.AdditionalTags["team"]).To(Equal("data-science"))

			Expect(machine.Spec.AdditionalMetadata["gpu-workload"]).To(Equal("training"))
			Expect(machine.Spec.AdditionalMetadata["framework"]).To(Equal("pytorch"))
			Expect(machine.Spec.AdditionalMetadata["dataset"]).To(Equal("imagenet"))
		})
	})

	Context("Error handling", func() {
		It("should handle missing SSH key gracefully", func() {
			By("Creating a machine with non-existent SSH key")
			testMachine = CreateDataCrunchMachine(machineName, namespace)
			testMachine.Spec.SSHKeyName = "non-existent-key"
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Waiting for error condition to be set")
			Eventually(func() bool {
				machine := &infrastructurev1beta1.DataCrunchMachine{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      machineName,
					Namespace: namespace,
				}, machine)
				if err != nil {
					return false
				}

				// Look for any condition with False status indicating an error
				for _, condition := range machine.Status.Conditions {
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

			By("Creating a machine without credentials")
			testMachine = CreateDataCrunchMachine(machineName, namespace)
			Expect(k8sClient.Create(ctx, testMachine)).To(Succeed())

			By("Verifying machine shows error condition due to missing credentials")
			Eventually(func() bool {
				machine := &infrastructurev1beta1.DataCrunchMachine{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      machineName,
					Namespace: namespace,
				}, machine)
				if err != nil {
					return false
				}

				// Should not be ready due to missing credentials
				return !machine.Status.Ready
			}, reconciliationTimeout, interval).Should(BeTrue())
		})
	})
})
