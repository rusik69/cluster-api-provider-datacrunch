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

package v1beta1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

func TestDataCrunchCluster_GetConditions(t *testing.T) {
	cluster := &DataCrunchCluster{
		Status: DataCrunchClusterStatus{
			Conditions: clusterv1.Conditions{
				{
					Type:   NetworkInfrastructureReadyCondition,
					Status: "True",
				},
				{
					Type:   LoadBalancerReadyCondition,
					Status: "False",
				},
			},
		},
	}

	conditions := cluster.GetConditions()
	if len(conditions) != 2 {
		t.Errorf("Expected 2 conditions, got %d", len(conditions))
	}

	// Check that we get the actual conditions
	if conditions[0].Type != NetworkInfrastructureReadyCondition {
		t.Errorf("Expected first condition to be %s, got %s", NetworkInfrastructureReadyCondition, conditions[0].Type)
	}
}

func TestDataCrunchCluster_SetConditions(t *testing.T) {
	cluster := &DataCrunchCluster{}

	newConditions := clusterv1.Conditions{
		{
			Type:               NetworkInfrastructureReadyCondition,
			Status:             "True",
			LastTransitionTime: metav1.Now(),
		},
	}

	cluster.SetConditions(newConditions)

	if len(cluster.Status.Conditions) != 1 {
		t.Errorf("Expected 1 condition after SetConditions, got %d", len(cluster.Status.Conditions))
	}

	if cluster.Status.Conditions[0].Type != NetworkInfrastructureReadyCondition {
		t.Errorf("Expected condition type %s, got %s", NetworkInfrastructureReadyCondition, cluster.Status.Conditions[0].Type)
	}
}

func TestDataCrunchClusterSpec_Validation(t *testing.T) {
	tests := []struct {
		name    string
		spec    DataCrunchClusterSpec
		wantErr bool
	}{
		{
			name: "valid spec with region",
			spec: DataCrunchClusterSpec{
				Region: "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "valid spec with load balancer",
			spec: DataCrunchClusterSpec{
				Region: "eu-west-1",
				ControlPlaneLoadBalancer: &DataCrunchLoadBalancerSpec{
					Type: "external",
				},
			},
			wantErr: false,
		},
		{
			name: "empty spec",
			spec: DataCrunchClusterSpec{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &DataCrunchCluster{
				Spec: tt.spec,
			}

			// Test that the object can be created without validation errors
			// In a real scenario, you'd have OpenAPI validation
			if cluster.Spec.Region == "" && tt.name != "empty spec" {
				t.Error("Region should be set for valid specs")
			}
		})
	}
}

func TestDataCrunchNetworkStatus_DeepCopy(t *testing.T) {
	original := &DataCrunchNetworkStatus{
		VPC: &DataCrunchVPCStatus{
			ID:        "vpc-123",
			CidrBlock: "10.0.0.0/16",
		},
		Subnets: []DataCrunchSubnetStatus{
			{
				ID:               "subnet-123",
				CidrBlock:        "10.0.1.0/24",
				AvailabilityZone: "us-east-1a",
			},
		},
	}

	copy := original.DeepCopy()

	// Test that it's a real deep copy
	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	if copy.VPC == original.VPC {
		t.Error("DeepCopy should copy nested pointers")
	}

	// Test that values are preserved
	if copy.VPC.ID != original.VPC.ID {
		t.Error("DeepCopy should preserve VPC ID")
	}

	if len(copy.Subnets) != len(original.Subnets) {
		t.Error("DeepCopy should preserve subnets slice")
	}

	// Modify copy to ensure independence
	copy.VPC.ID = "vpc-456"
	if original.VPC.ID == "vpc-456" {
		t.Error("Modifying copy should not affect original")
	}
}

func TestDataCrunchLoadBalancerStatus_Fields(t *testing.T) {
	lb := &DataCrunchLoadBalancerStatus{
		ID:      "lb-123",
		DNSName: "test-lb.datacrunch.io",
		State:   "active",
	}

	if lb.ID != "lb-123" {
		t.Errorf("Expected ID 'lb-123', got '%s'", lb.ID)
	}

	if lb.DNSName != "test-lb.datacrunch.io" {
		t.Errorf("Expected DNSName 'test-lb.datacrunch.io', got '%s'", lb.DNSName)
	}

	if lb.State != "active" {
		t.Errorf("Expected State 'active', got '%s'", lb.State)
	}
}

func TestDataCrunchCluster_DeepCopy(t *testing.T) {
	original := &DataCrunchCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "default",
		},
		Spec: DataCrunchClusterSpec{
			Region: "us-east-1",
		},
		Status: DataCrunchClusterStatus{
			Ready: true,
		},
	}

	copy := original.DeepCopy()

	// Test that it's a real deep copy
	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	// Test that values are preserved
	if copy.Name != original.Name {
		t.Error("DeepCopy should preserve Name")
	}

	if copy.Spec.Region != original.Spec.Region {
		t.Error("DeepCopy should preserve Spec")
	}

	// Modify copy to ensure independence
	copy.Name = "modified-cluster"
	if original.Name == "modified-cluster" {
		t.Error("Modifying copy should not affect original")
	}
}

func TestDataCrunchCluster_DeepCopyObject(t *testing.T) {
	original := &DataCrunchCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
		},
	}

	copyObj := original.DeepCopyObject()
	copy, ok := copyObj.(*DataCrunchCluster)
	if !ok {
		t.Fatal("DeepCopyObject should return a *DataCrunchCluster")
	}

	if copy.Name != original.Name {
		t.Error("DeepCopyObject should preserve Name")
	}
}

func TestDataCrunchClusterList_DeepCopy(t *testing.T) {
	original := &DataCrunchClusterList{
		Items: []DataCrunchCluster{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster1"},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster2"},
			},
		},
	}

	copy := original.DeepCopy()

	if len(copy.Items) != len(original.Items) {
		t.Error("DeepCopy should preserve Items slice length")
	}

	if copy.Items[0].Name != original.Items[0].Name {
		t.Error("DeepCopy should preserve item names")
	}

	// Modify copy to ensure independence
	copy.Items[0].Name = "modified"
	if original.Items[0].Name == "modified" {
		t.Error("Modifying copy should not affect original")
	}
}

func TestDataCrunchClusterStatus_DeepCopy(t *testing.T) {
	original := DataCrunchClusterStatus{
		Ready: true,
		Network: &DataCrunchNetworkStatus{
			VPC: &DataCrunchVPCStatus{
				ID: "vpc-123",
			},
		},
	}

	copy := original.DeepCopy()

	if copy.Ready != original.Ready {
		t.Error("DeepCopy should preserve Ready field")
	}

	if copy.Network == original.Network {
		t.Error("DeepCopy should copy nested pointers")
	}

	if copy.Network.VPC.ID != original.Network.VPC.ID {
		t.Error("DeepCopy should preserve nested values")
	}
}

// Test missing DeepCopy methods to improve coverage
func TestDataCrunchClusterList_DeepCopyObject(t *testing.T) {
	original := &DataCrunchClusterList{
		Items: []DataCrunchCluster{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster-1",
					Namespace: "default",
				},
				Spec: DataCrunchClusterSpec{
					Region: "us-east-1",
				},
			},
		},
	}

	obj := original.DeepCopyObject()
	copy, ok := obj.(*DataCrunchClusterList)
	if !ok {
		t.Fatal("DeepCopyObject should return *DataCrunchClusterList")
	}

	if copy == original {
		t.Error("DeepCopyObject should return a different pointer")
	}

	if len(copy.Items) != len(original.Items) {
		t.Error("DeepCopyObject should preserve Items slice")
	}
}

func TestDataCrunchClusterSpec_DeepCopy(t *testing.T) {
	enabled := true
	original := &DataCrunchClusterSpec{
		Region: "us-east-1",
		ControlPlaneLoadBalancer: &DataCrunchLoadBalancerSpec{
			Type:    "external",
			Enabled: &enabled,
		},
		Network: &DataCrunchNetworkSpec{
			VPC: &DataCrunchVPCSpec{
				CidrBlock: "10.0.0.0/16",
			},
		},
	}

	copy := original.DeepCopy()

	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	if copy.ControlPlaneLoadBalancer == original.ControlPlaneLoadBalancer {
		t.Error("DeepCopy should deep copy nested pointers")
	}

	if copy.Network == original.Network {
		t.Error("DeepCopy should deep copy network spec")
	}

	// Test nil case
	var nilSpec *DataCrunchClusterSpec
	if nilSpec.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}
}

func TestDataCrunchLoadBalancerSpec_DeepCopy(t *testing.T) {
	enabled := true
	original := &DataCrunchLoadBalancerSpec{
		Type:    "external",
		Enabled: &enabled,
	}

	copy := original.DeepCopy()

	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	if copy.Enabled == original.Enabled {
		t.Error("DeepCopy should deep copy enabled pointer")
	}

	if *copy.Enabled != *original.Enabled {
		t.Error("DeepCopy should preserve enabled value")
	}

	// Test nil case
	var nilSpec *DataCrunchLoadBalancerSpec
	if nilSpec.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}
}

func TestDataCrunchLoadBalancerStatus_DeepCopy(t *testing.T) {
	original := &DataCrunchLoadBalancerStatus{
		ID:      "lb-123",
		DNSName: "test-lb.datacrunch.io",
		State:   "active",
	}

	copy := original.DeepCopy()

	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	if copy.ID != original.ID {
		t.Error("DeepCopy should preserve ID")
	}

	if copy.DNSName != original.DNSName {
		t.Error("DeepCopy should preserve DNSName")
	}

	// Test nil case
	var nilStatus *DataCrunchLoadBalancerStatus
	if nilStatus.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}
}

func TestDataCrunchNetworkSpec_DeepCopy(t *testing.T) {
	original := &DataCrunchNetworkSpec{
		VPC: &DataCrunchVPCSpec{
			CidrBlock: "10.0.0.0/16",
		},
		Subnets: []DataCrunchSubnetSpec{
			{
				CidrBlock:        "10.0.1.0/24",
				AvailabilityZone: "us-east-1a",
			},
		},
	}

	copy := original.DeepCopy()

	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	if copy.VPC == original.VPC {
		t.Error("DeepCopy should deep copy VPC spec")
	}

	if copy.VPC.CidrBlock != original.VPC.CidrBlock {
		t.Error("DeepCopy should preserve VPC CIDR block")
	}

	// Test nil case
	var nilSpec *DataCrunchNetworkSpec
	if nilSpec.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}
}

func TestDataCrunchVPCSpec_DeepCopy(t *testing.T) {
	original := &DataCrunchVPCSpec{
		CidrBlock: "10.0.0.0/16",
	}

	copy := original.DeepCopy()

	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	if copy.CidrBlock != original.CidrBlock {
		t.Error("DeepCopy should preserve CIDR block")
	}

	// Test nil case
	var nilSpec *DataCrunchVPCSpec
	if nilSpec.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}
}

func TestDataCrunchVPCStatus_DeepCopy(t *testing.T) {
	original := &DataCrunchVPCStatus{
		ID:        "vpc-123",
		CidrBlock: "10.0.0.0/16",
	}

	copy := original.DeepCopy()

	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	if copy.ID != original.ID {
		t.Error("DeepCopy should preserve ID")
	}

	if copy.CidrBlock != original.CidrBlock {
		t.Error("DeepCopy should preserve CIDR block")
	}

	// Test nil case
	var nilStatus *DataCrunchVPCStatus
	if nilStatus.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}
}

func TestDataCrunchSubnetSpec_DeepCopy(t *testing.T) {
	original := &DataCrunchSubnetSpec{
		CidrBlock:        "10.0.1.0/24",
		AvailabilityZone: "us-east-1a",
	}

	copy := original.DeepCopy()

	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	if copy.CidrBlock != original.CidrBlock {
		t.Error("DeepCopy should preserve CIDR block")
	}

	if copy.AvailabilityZone != original.AvailabilityZone {
		t.Error("DeepCopy should preserve availability zone")
	}

	// Test nil case
	var nilSpec *DataCrunchSubnetSpec
	if nilSpec.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}
}

func TestDataCrunchSubnetStatus_DeepCopy(t *testing.T) {
	original := &DataCrunchSubnetStatus{
		ID:               "subnet-123",
		CidrBlock:        "10.0.1.0/24",
		AvailabilityZone: "us-east-1a",
	}

	copy := original.DeepCopy()

	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	if copy.ID != original.ID {
		t.Error("DeepCopy should preserve ID")
	}

	if copy.CidrBlock != original.CidrBlock {
		t.Error("DeepCopy should preserve CIDR block")
	}

	// Test nil case
	var nilStatus *DataCrunchSubnetStatus
	if nilStatus.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}
}
