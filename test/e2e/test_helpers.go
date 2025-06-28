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
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrastructurev1beta1 "github.com/rusik69/cluster-api-provider-datacrunch/api/v1beta1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// TestNamespace is the namespace used for tests
const TestNamespace = "default"

// TestClusterName is the default cluster name for tests
const TestClusterName = "test-cluster"

// TestMachineName is the default machine name for tests
const TestMachineName = "test-machine"

// CreateClusterWithInfra creates both a Cluster and DataCrunchCluster with proper ownership
func CreateClusterWithInfra(name, namespace string) (*clusterv1.Cluster, *infrastructurev1beta1.DataCrunchCluster) {
	// Create the Cluster first
	cluster := &clusterv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: clusterv1.ClusterSpec{
			InfrastructureRef: &corev1.ObjectReference{
				APIVersion: "infrastructure.cluster.x-k8s.io/v1beta1",
				Kind:       "DataCrunchCluster",
				Name:       name,
				Namespace:  namespace,
			},
		},
	}

	// Create the DataCrunchCluster with owner reference
	infraCluster := CreateDataCrunchCluster(name, namespace)
	infraCluster.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion:         "cluster.x-k8s.io/v1beta1",
			Kind:               "Cluster",
			Name:               name,
			UID:                cluster.UID, // This will be set when cluster is created
			Controller:         func() *bool { b := true; return &b }(),
			BlockOwnerDeletion: func() *bool { b := true; return &b }(),
		},
	})

	return cluster, infraCluster
}

// CreateDataCrunchCluster creates a test DataCrunchCluster resource
func CreateDataCrunchCluster(name, namespace string) *infrastructurev1beta1.DataCrunchCluster {
	cluster := &infrastructurev1beta1.DataCrunchCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: infrastructurev1beta1.DataCrunchClusterSpec{
			Region: "FIN-01",
			ControlPlaneEndpoint: clusterv1.APIEndpoint{
				Host: fmt.Sprintf("%s.datacrunch.local", name),
				Port: 6443,
			},
			ControlPlaneLoadBalancer: &infrastructurev1beta1.DataCrunchLoadBalancerSpec{
				Type: "external",
			},
			Network: &infrastructurev1beta1.DataCrunchNetworkSpec{
				VPC: &infrastructurev1beta1.DataCrunchVPCSpec{
					CidrBlock: "10.0.0.0/16",
				},
				Subnets: []infrastructurev1beta1.DataCrunchSubnetSpec{
					{
						CidrBlock:        "10.0.1.0/24",
						AvailabilityZone: "FIN-01a",
					},
				},
			},
		},
	}
	return cluster
}

// CreateDataCrunchMachine creates a test DataCrunchMachine resource
func CreateDataCrunchMachine(name, namespace string) *infrastructurev1beta1.DataCrunchMachine {
	machine := &infrastructurev1beta1.DataCrunchMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: infrastructurev1beta1.DataCrunchMachineSpec{
			InstanceType: "1xH100",
			Image:        "ubuntu-22.04-cuda-12.1",
			SSHKeyName:   "test-key",
			PublicIP:     func() *bool { b := true; return &b }(),
			RootVolume: &infrastructurev1beta1.Volume{
				Size: 100,
				Type: "fast-ssd",
			},
			AdditionalTags: map[string]string{
				"environment": "test",
				"project":     "e2e-testing",
			},
			AdditionalMetadata: map[string]string{
				"test": "true",
			},
		},
	}
	return machine
}

// CreateSecret creates a secret with DataCrunch credentials for testing
func CreateSecret(name, namespace string, mockAPIURL string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"clientID":     []byte("test-client-id"),
			"clientSecret": []byte("test-client-secret"),
			"apiURL":       []byte(mockAPIURL),
		},
	}
}

// WaitForClusterReady waits for a DataCrunchCluster to be in Ready state
func WaitForClusterReady(ctx context.Context, client client.Client, clusterName, namespace string, timeout time.Duration) {
	Eventually(func() bool {
		cluster := &infrastructurev1beta1.DataCrunchCluster{}
		err := client.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: namespace}, cluster)
		if err != nil {
			return false
		}
		return cluster.Status.Ready
	}, timeout, interval).Should(BeTrue())
}

// WaitForMachineReady waits for a DataCrunchMachine to be in Ready state
func WaitForMachineReady(ctx context.Context, client client.Client, machineName, namespace string, timeout time.Duration) {
	Eventually(func() bool {
		machine := &infrastructurev1beta1.DataCrunchMachine{}
		err := client.Get(ctx, types.NamespacedName{Name: machineName, Namespace: namespace}, machine)
		if err != nil {
			return false
		}
		return machine.Status.Ready
	}, timeout, interval).Should(BeTrue())
}

// WaitForMachineInstanceState waits for a DataCrunchMachine to reach a specific instance state
func WaitForMachineInstanceState(ctx context.Context, client client.Client, machineName, namespace string, expectedState infrastructurev1beta1.InstanceState, timeout time.Duration) {
	Eventually(func() bool {
		machine := &infrastructurev1beta1.DataCrunchMachine{}
		err := client.Get(ctx, types.NamespacedName{Name: machineName, Namespace: namespace}, machine)
		if err != nil {
			return false
		}
		return machine.Status.InstanceState != nil && *machine.Status.InstanceState == expectedState
	}, timeout, interval).Should(BeTrue())
}

// WaitForClusterCondition waits for a specific condition on a DataCrunchCluster
func WaitForClusterCondition(ctx context.Context, client client.Client, clusterName, namespace string, conditionType clusterv1.ConditionType, status corev1.ConditionStatus, timeout time.Duration) {
	Eventually(func() bool {
		cluster := &infrastructurev1beta1.DataCrunchCluster{}
		err := client.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: namespace}, cluster)
		if err != nil {
			return false
		}

		for _, condition := range cluster.Status.Conditions {
			if condition.Type == conditionType {
				return condition.Status == status
			}
		}
		return false
	}, timeout, interval).Should(BeTrue())
}

// WaitForMachineCondition waits for a specific condition on a DataCrunchMachine
func WaitForMachineCondition(ctx context.Context, client client.Client, machineName, namespace string, conditionType clusterv1.ConditionType, status corev1.ConditionStatus, timeout time.Duration) {
	Eventually(func() bool {
		machine := &infrastructurev1beta1.DataCrunchMachine{}
		err := client.Get(ctx, types.NamespacedName{Name: machineName, Namespace: namespace}, machine)
		if err != nil {
			return false
		}

		for _, condition := range machine.Status.Conditions {
			if condition.Type == conditionType {
				return condition.Status == status
			}
		}
		return false
	}, timeout, interval).Should(BeTrue())
}

// WaitForResourceDeletion waits for a resource to be deleted
func WaitForResourceDeletion(ctx context.Context, client client.Client, obj client.Object, timeout time.Duration) {
	Eventually(func() bool {
		err := client.Get(ctx, types.NamespacedName{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		}, obj)
		return err != nil
	}, timeout, interval).Should(BeTrue())
}

// GetClusterFromMachine retrieves the DataCrunchCluster associated with a machine
func GetClusterFromMachine(ctx context.Context, client client.Client, machine *infrastructurev1beta1.DataCrunchMachine) (*infrastructurev1beta1.DataCrunchCluster, error) {
	// In a real scenario, you'd get this from the Machine's owner references or labels
	// For testing, we'll assume the cluster has the same name prefix
	clusterName := machine.Name + "-cluster"

	cluster := &infrastructurev1beta1.DataCrunchCluster{}
	err := client.Get(ctx, types.NamespacedName{
		Name:      clusterName,
		Namespace: machine.Namespace,
	}, cluster)

	return cluster, err
}

// GetMachineProviderID extracts the provider ID from a machine's status
func GetMachineProviderID(machine *infrastructurev1beta1.DataCrunchMachine) string {
	if machine.Spec.ProviderID == nil {
		return ""
	}
	return *machine.Spec.ProviderID
}

// HasCondition checks if a resource has a specific condition
func HasCondition(conditions clusterv1.Conditions, conditionType clusterv1.ConditionType) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return true
		}
	}
	return false
}

// GetCondition retrieves a specific condition from a list
func GetCondition(conditions clusterv1.Conditions, conditionType clusterv1.ConditionType) *clusterv1.Condition {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

// SetupDataCrunchCredentials sets up environment variables for DataCrunch API access
func SetupDataCrunchCredentials(mockAPIURL string) {
	os.Setenv("DATACRUNCH_CLIENT_ID", "test-client-id")
	os.Setenv("DATACRUNCH_CLIENT_SECRET", "test-client-secret")
	os.Setenv("DATACRUNCH_API_URL", mockAPIURL)
}

// CleanupDataCrunchCredentials removes DataCrunch environment variables
func CleanupDataCrunchCredentials() {
	os.Unsetenv("DATACRUNCH_CLIENT_ID")
	os.Unsetenv("DATACRUNCH_CLIENT_SECRET")
	os.Unsetenv("DATACRUNCH_API_URL")
}

// CreateTestEnvironment creates a complete test environment with cluster and machine
func CreateTestEnvironment(ctx context.Context, client client.Client, mockAPIURL string) (*infrastructurev1beta1.DataCrunchCluster, *infrastructurev1beta1.DataCrunchMachine, *corev1.Secret) {
	// Create secret for credentials
	secret := CreateSecret("datacrunch-credentials", TestNamespace, mockAPIURL)
	Expect(client.Create(ctx, secret)).To(Succeed())

	// Create cluster
	cluster := CreateDataCrunchCluster(TestClusterName, TestNamespace)
	Expect(client.Create(ctx, cluster)).To(Succeed())

	// Create machine
	machine := CreateDataCrunchMachine(TestMachineName, TestNamespace)
	Expect(client.Create(ctx, machine)).To(Succeed())

	return cluster, machine, secret
}

// CleanupTestEnvironment removes all test resources
func CleanupTestEnvironment(ctx context.Context, client client.Client, cluster *infrastructurev1beta1.DataCrunchCluster, machine *infrastructurev1beta1.DataCrunchMachine, secret *corev1.Secret) {
	// Delete machine first
	if machine != nil {
		Expect(client.Delete(ctx, machine)).To(Succeed())
		WaitForResourceDeletion(ctx, client, machine, timeout)
	}

	// Delete cluster
	if cluster != nil {
		Expect(client.Delete(ctx, cluster)).To(Succeed())
		WaitForResourceDeletion(ctx, client, cluster, timeout)
	}

	// Delete secret
	if secret != nil {
		Expect(client.Delete(ctx, secret)).To(Succeed())
		WaitForResourceDeletion(ctx, client, secret, timeout)
	}
}
