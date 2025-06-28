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

func TestDataCrunchMachine_GetConditions(t *testing.T) {
	machine := &DataCrunchMachine{
		Status: DataCrunchMachineStatus{
			Conditions: clusterv1.Conditions{
				{
					Type:   InstanceReadyCondition,
					Status: "True",
				},
				{
					Type:   "CustomCondition",
					Status: "False",
				},
			},
		},
	}

	conditions := machine.GetConditions()
	if len(conditions) != 2 {
		t.Errorf("Expected 2 conditions, got %d", len(conditions))
	}

	if conditions[0].Type != InstanceReadyCondition {
		t.Errorf("Expected first condition to be %s, got %s", InstanceReadyCondition, conditions[0].Type)
	}
}

func TestDataCrunchMachine_SetConditions(t *testing.T) {
	machine := &DataCrunchMachine{}

	newConditions := clusterv1.Conditions{
		{
			Type:               InstanceReadyCondition,
			Status:             "False",
			LastTransitionTime: metav1.Now(),
			Reason:             "InstanceNotFound",
			Message:            "Instance not found in DataCrunch",
		},
	}

	machine.SetConditions(newConditions)

	if len(machine.Status.Conditions) != 1 {
		t.Errorf("Expected 1 condition after SetConditions, got %d", len(machine.Status.Conditions))
	}

	condition := machine.Status.Conditions[0]
	if condition.Type != InstanceReadyCondition {
		t.Errorf("Expected condition type %s, got %s", InstanceReadyCondition, condition.Type)
	}

	if condition.Reason != "InstanceNotFound" {
		t.Errorf("Expected condition reason 'InstanceNotFound', got '%s'", condition.Reason)
	}
}

func TestDataCrunchMachineSpec_InstanceTypes(t *testing.T) {
	tests := []struct {
		name         string
		instanceType string
		expectValid  bool
	}{
		{"H100 instance", "1xH100.80G", true},
		{"H200 instance", "8xH200.141G", true},
		{"A100 instance", "1xA100.40G", true},
		{"CPU instance", "16VCPU.64G", true},
		{"Invalid instance", "invalid-type", true}, // We don't validate format in types
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := DataCrunchMachineSpec{
				InstanceType: tt.instanceType,
				Image:        "ubuntu-20.04",
			}

			machine := &DataCrunchMachine{
				Spec: spec,
			}

			if machine.Spec.InstanceType != tt.instanceType {
				t.Errorf("Expected instance type %s, got %s", tt.instanceType, machine.Spec.InstanceType)
			}
		})
	}
}

func TestDataCrunchMachineSpec_SSHKey(t *testing.T) {
	spec := DataCrunchMachineSpec{
		InstanceType: "1xA100.40G",
		Image:        "ubuntu-20.04",
		SSHKeyName:   "my-ssh-key",
	}

	if spec.SSHKeyName != "my-ssh-key" {
		t.Errorf("Expected SSH key name 'my-ssh-key', got '%s'", spec.SSHKeyName)
	}
}

func TestDataCrunchMachineStatus_Ready(t *testing.T) {
	state := InstanceStateRunning
	status := DataCrunchMachineStatus{
		Ready:         true,
		InstanceState: &state,
	}

	if !status.Ready {
		t.Error("Expected Ready to be true")
	}

	if *status.InstanceState != InstanceStateRunning {
		t.Errorf("Expected instance state 'running', got '%s'", *status.InstanceState)
	}
}

func TestDataCrunchMachineStatus_Addresses(t *testing.T) {
	status := DataCrunchMachineStatus{
		Addresses: []clusterv1.MachineAddress{
			{
				Type:    clusterv1.MachineExternalIP,
				Address: "1.2.3.4",
			},
			{
				Type:    clusterv1.MachineInternalIP,
				Address: "10.0.0.10",
			},
			{
				Type:    clusterv1.MachineHostName,
				Address: "instance-1.datacrunch.local",
			},
		},
	}

	if len(status.Addresses) != 3 {
		t.Errorf("Expected 3 addresses, got %d", len(status.Addresses))
	}

	// Test external IP
	externalIP := status.Addresses[0]
	if externalIP.Type != clusterv1.MachineExternalIP {
		t.Errorf("Expected first address type to be ExternalIP, got %s", externalIP.Type)
	}
	if externalIP.Address != "1.2.3.4" {
		t.Errorf("Expected external IP '1.2.3.4', got '%s'", externalIP.Address)
	}

	// Test internal IP
	internalIP := status.Addresses[1]
	if internalIP.Type != clusterv1.MachineInternalIP {
		t.Errorf("Expected second address type to be InternalIP, got %s", internalIP.Type)
	}
	if internalIP.Address != "10.0.0.10" {
		t.Errorf("Expected internal IP '10.0.0.10', got '%s'", internalIP.Address)
	}
}

func TestDataCrunchMachineStatus_FailureHandling(t *testing.T) {
	message := "Failed to create instance due to quota exceeded"

	status := DataCrunchMachineStatus{
		FailureMessage: &message,
		Ready:          false,
	}

	if status.FailureMessage == nil {
		t.Error("Expected FailureMessage to be set")
	} else if *status.FailureMessage != message {
		t.Errorf("Expected failure message '%s', got '%s'", message, *status.FailureMessage)
	}

	if status.Ready {
		t.Error("Expected Ready to be false when there's a failure")
	}
}

func TestDataCrunchMachineSpec_DeepCopy(t *testing.T) {
	original := &DataCrunchMachineSpec{
		InstanceType: "1xH100.80G",
		Image:        "ubuntu-22.04",
		SSHKeyName:   "my-key",
		AdditionalTags: map[string]string{
			"environment": "test",
			"project":     "cluster-api",
		},
	}

	copy := original.DeepCopy()

	// Test that it's a real deep copy
	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	// Test that maps are copied
	if &copy.AdditionalTags == &original.AdditionalTags {
		t.Error("DeepCopy should copy map")
	}

	// Modify copy to ensure independence
	copy.InstanceType = "2xA100.80G"
	copy.SSHKeyName = "modified-key"
	copy.AdditionalTags["environment"] = "modified"

	if original.InstanceType == "2xA100.80G" {
		t.Error("Modifying copy should not affect original InstanceType")
	}

	if original.SSHKeyName == "modified-key" {
		t.Error("Modifying copy should not affect original SSHKeyName")
	}

	if original.AdditionalTags["environment"] == "modified" {
		t.Error("Modifying copy should not affect original AdditionalTags")
	}
}

func TestDataCrunchMachine_DeepCopy(t *testing.T) {
	original := &DataCrunchMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-machine",
			Namespace: "default",
		},
		Spec: DataCrunchMachineSpec{
			InstanceType: "1xH100.80G",
			Image:        "ubuntu-20.04",
		},
		Status: DataCrunchMachineStatus{
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

	if copy.Spec.InstanceType != original.Spec.InstanceType {
		t.Error("DeepCopy should preserve Spec")
	}

	// Modify copy to ensure independence
	copy.Name = "modified-machine"
	if original.Name == "modified-machine" {
		t.Error("Modifying copy should not affect original")
	}
}

func TestDataCrunchMachine_DeepCopyObject(t *testing.T) {
	original := &DataCrunchMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-machine",
		},
	}

	copyObj := original.DeepCopyObject()
	copy, ok := copyObj.(*DataCrunchMachine)
	if !ok {
		t.Fatal("DeepCopyObject should return a *DataCrunchMachine")
	}

	if copy.Name != original.Name {
		t.Error("DeepCopyObject should preserve Name")
	}
}

func TestDataCrunchMachineList_DeepCopy(t *testing.T) {
	original := &DataCrunchMachineList{
		Items: []DataCrunchMachine{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "machine1"},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "machine2"},
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

func TestInstanceState_Constants(t *testing.T) {
	tests := []struct {
		name     string
		state    InstanceState
		expected string
	}{
		{"pending state", InstanceStatePending, "pending"},
		{"running state", InstanceStateRunning, "running"},
		{"shutting down state", InstanceStateShuttingDown, "shutting-down"},
		{"terminated state", InstanceStateTerminated, "terminated"},
		{"stopping state", InstanceStateStopping, "stopping"},
		{"stopped state", InstanceStateStopped, "stopped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.state) != tt.expected {
				t.Errorf("Expected state '%s', got '%s'", tt.expected, string(tt.state))
			}
		})
	}
}

func TestSpotMachineOptions_Fields(t *testing.T) {
	maxPrice := "0.50"
	spot := &SpotMachineOptions{
		MaxPrice: &maxPrice,
	}

	if spot.MaxPrice == nil {
		t.Error("Expected MaxPrice to be set")
	} else if *spot.MaxPrice != maxPrice {
		t.Errorf("Expected MaxPrice '%s', got '%s'", maxPrice, *spot.MaxPrice)
	}
}

func TestVolume_Fields(t *testing.T) {
	encrypted := true
	iops := int64(1000)

	volume := &Volume{
		Size:      100,
		Type:      "SSD",
		Encrypted: &encrypted,
		IOPS:      &iops,
	}

	if volume.Size != 100 {
		t.Errorf("Expected Size 100, got %d", volume.Size)
	}

	if volume.Type != "SSD" {
		t.Errorf("Expected Type 'SSD', got '%s'", volume.Type)
	}

	if volume.Encrypted == nil || !*volume.Encrypted {
		t.Error("Expected Encrypted to be true")
	}

	if volume.IOPS == nil || *volume.IOPS != 1000 {
		t.Error("Expected IOPS to be 1000")
	}
}

// Test missing DeepCopy methods to improve coverage
func TestDataCrunchMachineList_DeepCopyObject(t *testing.T) {
	original := &DataCrunchMachineList{
		Items: []DataCrunchMachine{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine-1",
					Namespace: "default",
				},
				Spec: DataCrunchMachineSpec{
					InstanceType: "1xH100.80G",
					Image:        "ubuntu-20.04",
				},
			},
		},
	}

	obj := original.DeepCopyObject()
	copy, ok := obj.(*DataCrunchMachineList)
	if !ok {
		t.Fatal("DeepCopyObject should return *DataCrunchMachineList")
	}

	if copy == original {
		t.Error("DeepCopyObject should return a different pointer")
	}

	if len(copy.Items) != len(original.Items) {
		t.Error("DeepCopyObject should preserve Items slice")
	}
}

func TestDataCrunchMachineStatus_DeepCopy(t *testing.T) {
	state := InstanceStateRunning
	message := "Instance is running"
	original := &DataCrunchMachineStatus{
		Ready:          true,
		InstanceState:  &state,
		FailureMessage: &message,
		Addresses: []clusterv1.MachineAddress{
			{
				Type:    clusterv1.MachineExternalIP,
				Address: "1.2.3.4",
			},
		},
	}

	copy := original.DeepCopy()

	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	if copy.Ready != original.Ready {
		t.Error("DeepCopy should preserve Ready field")
	}

	if copy.InstanceState == original.InstanceState {
		t.Error("DeepCopy should deep copy instance state pointer")
	}

	if *copy.InstanceState != *original.InstanceState {
		t.Error("DeepCopy should preserve instance state value")
	}

	if copy.FailureMessage == original.FailureMessage {
		t.Error("DeepCopy should deep copy failure message pointer")
	}

	// Test nil case
	var nilStatus *DataCrunchMachineStatus
	if nilStatus.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}
}

func TestDataCrunchMachineSpec_DeepCopy_CompleteTest(t *testing.T) {
	providerID := "datacrunch://instance-123"
	uncompressed := true
	maxPrice := "0.5"
	original := &DataCrunchMachineSpec{
		InstanceType:         "1xH100.80G",
		Image:                "ubuntu-20.04",
		ProviderID:           &providerID,
		SSHKeyName:           "my-key",
		UncompressedUserData: &uncompressed,
		AdditionalMetadata: map[string]string{
			"key1": "value1",
		},
		AdditionalTags: map[string]string{
			"Environment": "test",
		},
		RootVolume: &Volume{
			Size: 50,
			Type: "ssd",
		},
		NetworkInterfaces: []NetworkInterface{
			{
				SubnetID: "subnet-123",
			},
		},
		Spot: &SpotMachineOptions{
			MaxPrice: &maxPrice,
		},
	}

	copy := original.DeepCopy()

	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	if copy.ProviderID == original.ProviderID {
		t.Error("DeepCopy should deep copy ProviderID pointer")
	}

	if *copy.ProviderID != *original.ProviderID {
		t.Error("DeepCopy should preserve ProviderID value")
	}

	if copy.RootVolume == original.RootVolume {
		t.Error("DeepCopy should deep copy RootVolume")
	}

	// Test nil case
	var nilSpec *DataCrunchMachineSpec
	if nilSpec.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}
}

func TestSpotMachineOptions_DeepCopy(t *testing.T) {
	maxPrice := "0.5"
	original := &SpotMachineOptions{
		MaxPrice: &maxPrice,
	}

	copy := original.DeepCopy()

	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	if copy.MaxPrice == original.MaxPrice {
		t.Error("DeepCopy should deep copy MaxPrice pointer")
	}

	if *copy.MaxPrice != *original.MaxPrice {
		t.Error("DeepCopy should preserve MaxPrice value")
	}

	// Test nil case
	var nilOptions *SpotMachineOptions
	if nilOptions.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}
}

func TestVolume_DeepCopy(t *testing.T) {
	original := &Volume{
		Size: 100,
		Type: "ssd",
	}

	copy := original.DeepCopy()

	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	if copy.Size != original.Size {
		t.Error("DeepCopy should preserve Size")
	}

	if copy.Type != original.Type {
		t.Error("DeepCopy should preserve Type")
	}

	// Test nil case
	var nilVolume *Volume
	if nilVolume.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}
}

func TestNetworkInterface_DeepCopy(t *testing.T) {
	associatePublicIP := true
	original := &NetworkInterface{
		SubnetID:                 "subnet-123",
		AssociatePublicIPAddress: &associatePublicIP,
	}

	copy := original.DeepCopy()

	if copy == original {
		t.Error("DeepCopy should return a different pointer")
	}

	if copy.SubnetID != original.SubnetID {
		t.Error("DeepCopy should preserve SubnetID")
	}

	if copy.AssociatePublicIPAddress == original.AssociatePublicIPAddress {
		t.Error("DeepCopy should deep copy AssociatePublicIPAddress pointer")
	}

	// Test nil case
	var nilInterface *NetworkInterface
	if nilInterface.DeepCopy() != nil {
		t.Error("DeepCopy of nil should return nil")
	}
}

func TestDataCrunchMachineSpec_AdditionalFields(t *testing.T) {
	spec := DataCrunchMachineSpec{
		InstanceType: "1xA100.40G",
		Image:        "ubuntu-20.04",
		AdditionalMetadata: map[string]string{
			"project":     "test-project",
			"environment": "dev",
		},
		AdditionalTags: map[string]string{
			"Owner": "team-ai",
			"Cost":  "research",
		},
	}

	// Test metadata
	if len(spec.AdditionalMetadata) != 2 {
		t.Errorf("Expected 2 metadata entries, got %d", len(spec.AdditionalMetadata))
	}

	if spec.AdditionalMetadata["project"] != "test-project" {
		t.Error("Metadata should preserve project value")
	}

	// Test tags
	if len(spec.AdditionalTags) != 2 {
		t.Errorf("Expected 2 tag entries, got %d", len(spec.AdditionalTags))
	}

	if spec.AdditionalTags["Owner"] != "team-ai" {
		t.Error("Tags should preserve Owner value")
	}
}

func TestDataCrunchMachineSpec_NetworkInterfaces(t *testing.T) {
	associatePublicIP1 := true
	associatePublicIP2 := false
	spec := DataCrunchMachineSpec{
		InstanceType: "1xA100.40G",
		Image:        "ubuntu-20.04",
		NetworkInterfaces: []NetworkInterface{
			{
				SubnetID:                 "subnet-123",
				AssociatePublicIPAddress: &associatePublicIP1,
			},
			{
				SubnetID:                 "subnet-456",
				AssociatePublicIPAddress: &associatePublicIP2,
			},
		},
	}

	if len(spec.NetworkInterfaces) != 2 {
		t.Errorf("Expected 2 network interfaces, got %d", len(spec.NetworkInterfaces))
	}

	// Test first interface
	firstInterface := spec.NetworkInterfaces[0]
	if firstInterface.SubnetID != "subnet-123" {
		t.Error("First interface should have correct subnet ID")
	}

	if !*firstInterface.AssociatePublicIPAddress {
		t.Error("First interface should associate public IP")
	}

	// Test second interface
	secondInterface := spec.NetworkInterfaces[1]
	if *secondInterface.AssociatePublicIPAddress {
		t.Error("Second interface should not associate public IP")
	}
}
