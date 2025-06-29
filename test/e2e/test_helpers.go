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

	"github.com/onsi/gomega"
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
	gomega.Eventually(func() bool {
		cluster := &infrastructurev1beta1.DataCrunchCluster{}
		err := client.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: namespace}, cluster)
		if err != nil {
			return false
		}
		return cluster.Status.Ready
	}, timeout, interval).Should(gomega.BeTrue())
}

// WaitForMachineReady waits for a DataCrunchMachine to be in Ready state
func WaitForMachineReady(ctx context.Context, client client.Client, machineName, namespace string, timeout time.Duration) {
	gomega.Eventually(func() bool {
		machine := &infrastructurev1beta1.DataCrunchMachine{}
		err := client.Get(ctx, types.NamespacedName{Name: machineName, Namespace: namespace}, machine)
		if err != nil {
			return false
		}
		return machine.Status.Ready
	}, timeout, interval).Should(gomega.BeTrue())
}

// WaitForMachineInstanceState waits for a DataCrunchMachine to reach a specific instance state
func WaitForMachineInstanceState(ctx context.Context, client client.Client, machineName, namespace string, expectedState infrastructurev1beta1.InstanceState, timeout time.Duration) {
	gomega.Eventually(func() bool {
		machine := &infrastructurev1beta1.DataCrunchMachine{}
		err := client.Get(ctx, types.NamespacedName{Name: machineName, Namespace: namespace}, machine)
		if err != nil {
			return false
		}
		return machine.Status.InstanceState != nil && *machine.Status.InstanceState == expectedState
	}, timeout, interval).Should(gomega.BeTrue())
}

// WaitForClusterCondition waits for a specific condition on a DataCrunchCluster
func WaitForClusterCondition(ctx context.Context, client client.Client, clusterName, namespace string, conditionType clusterv1.ConditionType, status corev1.ConditionStatus, timeout time.Duration) {
	gomega.Eventually(func() bool {
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
	}, timeout, interval).Should(gomega.BeTrue())
}

// WaitForMachineCondition waits for a specific condition on a DataCrunchMachine
func WaitForMachineCondition(ctx context.Context, client client.Client, machineName, namespace string, conditionType clusterv1.ConditionType, status corev1.ConditionStatus, timeout time.Duration) {
	gomega.Eventually(func() bool {
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
	}, timeout, interval).Should(gomega.BeTrue())
}

// WaitForResourceDeletion waits for a resource to be deleted
func WaitForResourceDeletion(ctx context.Context, client client.Client, obj client.Object, timeout time.Duration) {
	gomega.Eventually(func() bool {
		err := client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
		return err != nil // Resource should not be found (deleted)
	}, timeout, interval).Should(gomega.BeTrue())
}

// GetClusterFromMachine gets the cluster associated with a machine
func GetClusterFromMachine(ctx context.Context, client client.Client, machine *infrastructurev1beta1.DataCrunchMachine) (*infrastructurev1beta1.DataCrunchCluster, error) {
	// Get the owner cluster name from the machine's labels/annotations
	clusterName := machine.Labels[clusterv1.ClusterNameLabel]
	if clusterName == "" {
		return nil, fmt.Errorf("machine %s/%s does not have cluster label", machine.Namespace, machine.Name)
	}

	cluster := &infrastructurev1beta1.DataCrunchCluster{}
	err := client.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: machine.Namespace}, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster %s/%s: %w", machine.Namespace, clusterName, err)
	}

	return cluster, nil
}

// GetMachineProviderID returns the provider ID for a machine
func GetMachineProviderID(machine *infrastructurev1beta1.DataCrunchMachine) string {
	if machine.Spec.ProviderID != nil {
		return *machine.Spec.ProviderID
	}
	return ""
}

// HasCondition checks if a condition type exists in a list of conditions
func HasCondition(conditions clusterv1.Conditions, conditionType clusterv1.ConditionType) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return true
		}
	}
	return false
}

// GetCondition returns a specific condition from a list of conditions
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
	_ = os.Setenv("DATACRUNCH_CLIENT_ID", "test-client-id")
	_ = os.Setenv("DATACRUNCH_CLIENT_SECRET", "test-client-secret")
	_ = os.Setenv("DATACRUNCH_API_URL", mockAPIURL)
}

// CleanupDataCrunchCredentials removes DataCrunch environment variables
func CleanupDataCrunchCredentials() {
	_ = os.Unsetenv("DATACRUNCH_CLIENT_ID")
	_ = os.Unsetenv("DATACRUNCH_CLIENT_SECRET")
	_ = os.Unsetenv("DATACRUNCH_API_URL")
}

// CreateTestEnvironment creates a complete test environment with cluster, machine, and secret
func CreateTestEnvironment(ctx context.Context, client client.Client, mockAPIURL string) (*infrastructurev1beta1.DataCrunchCluster, *infrastructurev1beta1.DataCrunchMachine, *corev1.Secret) {
	cluster := CreateDataCrunchCluster(TestClusterName, TestNamespace)
	machine := CreateDataCrunchMachine(TestMachineName, TestNamespace)
	secret := CreateSecret("datacrunch-credentials", TestNamespace, mockAPIURL)

	// Create resources
	gomega.Expect(client.Create(ctx, cluster)).To(gomega.Succeed())
	gomega.Expect(client.Create(ctx, machine)).To(gomega.Succeed())
	gomega.Expect(client.Create(ctx, secret)).To(gomega.Succeed())

	return cluster, machine, secret
}

// CleanupTestEnvironment cleans up test resources
func CleanupTestEnvironment(ctx context.Context, client client.Client, cluster *infrastructurev1beta1.DataCrunchCluster, machine *infrastructurev1beta1.DataCrunchMachine, secret *corev1.Secret) {
	// Delete resources (ignore errors as they might already be deleted)
	_ = client.Delete(ctx, machine)
	_ = client.Delete(ctx, cluster)
	_ = client.Delete(ctx, secret)

	// Wait for deletion
	WaitForResourceDeletion(ctx, client, machine, timeout)
	WaitForResourceDeletion(ctx, client, cluster, timeout)
	WaitForResourceDeletion(ctx, client, secret, timeout)
}
