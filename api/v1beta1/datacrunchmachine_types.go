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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
)

const (
	// MachineFinalizer allows DataCrunchMachineReconciler to clean up DataCrunch resources associated with DataCrunchMachine before
	// removing it from the apiserver.
	MachineFinalizer = "datacrunchmachine.infrastructure.cluster.x-k8s.io"
)

// DataCrunchMachineSpec defines the desired state of DataCrunchMachine
type DataCrunchMachineSpec struct {
	// InstanceType specifies the DataCrunch instance type (e.g., "1V100.6V", "1H100.80S.32V", "8H100.80S.176V")
	InstanceType string `json:"instanceType"`

	// Image specifies the image to use for the instance
	// +optional
	Image string `json:"image,omitempty"`

	// SSHKeyName specifies the SSH key name to use for the instance
	// +optional
	SSHKeyName string `json:"sshKeyName,omitempty"`

	// ProviderID is the unique identifier as specified by the cloud provider.
	// +optional
	ProviderID *string `json:"providerID,omitempty"`

	// AdditionalMetadata is the additional metadata for the machine
	// +optional
	AdditionalMetadata map[string]string `json:"additionalMetadata,omitempty"`

	// AdditionalTags is an optional set of tags to add to an instance, in addition to the ones added by default by the
	// DataCrunch provider. Tags must be compliant with DataCrunch's tag naming conventions.
	// +optional
	AdditionalTags map[string]string `json:"additionalTags,omitempty"`

	// RootVolume encapsulates the configuration options for the root volume
	// +optional
	RootVolume *Volume `json:"rootVolume,omitempty"`

	// UncompressedUserData specifies whether the user data is compressed or not.
	// +optional
	UncompressedUserData *bool `json:"uncompressedUserData,omitempty"`

	// NetworkInterfaces specifies a list of network interfaces to attach to the instance
	// +optional
	NetworkInterfaces []NetworkInterface `json:"networkInterfaces,omitempty"`

	// PublicIP specifies whether the instance should get a public IP
	// +optional
	PublicIP *bool `json:"publicIP,omitempty"`

	// Spot configures the instance to use spot pricing
	// +optional
	Spot *SpotMachineOptions `json:"spot,omitempty"`
}

// SpotMachineOptions defines the configuration for spot instances
type SpotMachineOptions struct {
	// MaxPrice is the maximum price you're willing to pay for the instance
	// If not specified, the on-demand price is used as the maximum
	// +optional
	MaxPrice *string `json:"maxPrice,omitempty"`
}

// Volume encapsulates the configuration options for the storage device
type Volume struct {
	// Size specifies the size of the storage device in GB
	// +optional
	Size int64 `json:"size,omitempty"`

	// Type is the type of storage to use (e.g., "SSD", "HDD")
	// +optional
	Type string `json:"type,omitempty"`

	// Encrypted is whether the volume should be encrypted
	// +optional
	Encrypted *bool `json:"encrypted,omitempty"`

	// IOPS is the number of IOPS for the storage device
	// +optional
	IOPS *int64 `json:"iops,omitempty"`
}

// NetworkInterface defines the network interface configuration
type NetworkInterface struct {
	// SubnetID is the ID of the subnet to use
	// +optional
	SubnetID string `json:"subnetID,omitempty"`

	// DeviceIndex is the device index for the network interface
	// +optional
	DeviceIndex *int64 `json:"deviceIndex,omitempty"`

	// AssociatePublicIPAddress specifies whether to associate a public IP address
	// +optional
	AssociatePublicIPAddress *bool `json:"associatePublicIPAddress,omitempty"`

	// DeleteOnTermination specifies whether to delete the network interface on termination
	// +optional
	DeleteOnTermination *bool `json:"deleteOnTermination,omitempty"`

	// SecondaryPrivateIPAddressCount is the number of secondary private IP addresses
	// +optional
	SecondaryPrivateIPAddressCount *int64 `json:"secondaryPrivateIPAddressCount,omitempty"`

	// SecurityGroupIDs is a list of security group IDs to associate with the network interface
	// +optional
	SecurityGroupIDs []string `json:"securityGroupIDs,omitempty"`
}

// DataCrunchMachineStatus defines the observed state of DataCrunchMachine
type DataCrunchMachineStatus struct {
	// Ready denotes that the machine (infrastructure) is ready.
	// +optional
	Ready bool `json:"ready"`

	// Addresses contains the DataCrunch instance associated addresses.
	// +optional
	Addresses []clusterv1.MachineAddress `json:"addresses,omitempty"`

	// InstanceState is the current state of the DataCrunch instance for this machine.
	// +optional
	InstanceState *InstanceState `json:"instanceState,omitempty"`

	// Conditions defines current service state of the DataCrunchMachine.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	// FailureReason will be set in the event that there is a terminal problem
	// reconciling the machine and will contain a succinct value suitable
	// for machine interpretation.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureReason *capierrors.MachineStatusError `json:"failureReason,omitempty"`

	// FailureMessage will be set in the event that there is a terminal problem
	// reconciling the machine and will contain a more verbose string suitable
	// for logging and human consumption.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// InterruptionReason contains the interrupt action reason
	// +optional
	InterruptionReason *string `json:"interruptionReason,omitempty"`
}

// InstanceState describes the state of a DataCrunch instance.
type InstanceState string

const (
	// InstanceStatePending is the string representing an instance in pending state
	InstanceStatePending = InstanceState("pending")

	// InstanceStateRunning is the string representing an instance in running state
	InstanceStateRunning = InstanceState("running")

	// InstanceStateShuttingDown is the string representing an instance shutting down
	InstanceStateShuttingDown = InstanceState("shutting-down")

	// InstanceStateTerminated is the string representing an instance that has been terminated
	InstanceStateTerminated = InstanceState("terminated")

	// InstanceStateStopping is the string representing an instance
	// that is in the process of being stopped and can be restarted
	InstanceStateStopping = InstanceState("stopping")

	// InstanceStateStopped is the string representing an instance
	// that has been stopped and can be restarted
	InstanceStateStopped = InstanceState("stopped")
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this DataCrunchMachine belongs"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.instanceState",description="DataCrunch instance state"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Machine ready status"
// +kubebuilder:printcolumn:name="InstanceID",type="string",JSONPath=".spec.providerID",description="DataCrunch instance ID"
// +kubebuilder:printcolumn:name="Machine",type="string",JSONPath=".metadata.ownerReferences[?(@.kind==\"Machine\")].name",description="Machine object which owns with this DataCrunchMachine"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of DataCrunchMachine"

// DataCrunchMachine is the Schema for the datacrunchmachines API
type DataCrunchMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataCrunchMachineSpec   `json:"spec,omitempty"`
	Status DataCrunchMachineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DataCrunchMachineList contains a list of DataCrunchMachine
type DataCrunchMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataCrunchMachine `json:"items"`
}

// GetConditions returns the observations of the operational state of the DataCrunchMachine resource.
func (m *DataCrunchMachine) GetConditions() clusterv1.Conditions {
	return m.Status.Conditions
}

// SetConditions sets the underlying service state of the DataCrunchMachine to the predicate provided.
func (m *DataCrunchMachine) SetConditions(conditions clusterv1.Conditions) {
	m.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&DataCrunchMachine{}, &DataCrunchMachineList{})
}
