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
)

const (
	// ClusterFinalizer allows DataCrunchClusterReconciler to clean up DataCrunch resources associated with DataCrunchCluster before
	// removing it from the apiserver.
	ClusterFinalizer = "datacrunchcluster.infrastructure.cluster.x-k8s.io"
)

// DataCrunchClusterSpec defines the desired state of DataCrunchCluster
type DataCrunchClusterSpec struct {
	// Region is the DataCrunch region where the cluster will be created
	// +optional
	Region string `json:"region,omitempty"`

	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint,omitempty"`

	// ControlPlaneLoadBalancer configures the optional control plane load balancer.
	// +optional
	ControlPlaneLoadBalancer *DataCrunchLoadBalancerSpec `json:"controlPlaneLoadBalancer,omitempty"`

	// Network configuration for the cluster
	// +optional
	Network *DataCrunchNetworkSpec `json:"network,omitempty"`
}

// DataCrunchLoadBalancerSpec defines the load balancer configuration
type DataCrunchLoadBalancerSpec struct {
	// Enabled specifies whether to create a load balancer for the control plane
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Type specifies the type of load balancer
	// +optional
	Type string `json:"type,omitempty"`

	// HealthCheckPath is the path for health checks
	// +optional
	HealthCheckPath string `json:"healthCheckPath,omitempty"`
}

// DataCrunchNetworkSpec defines network configuration
type DataCrunchNetworkSpec struct {
	// VPC specifies the VPC configuration
	// +optional
	VPC *DataCrunchVPCSpec `json:"vpc,omitempty"`

	// Subnets specifies the subnet configurations
	// +optional
	Subnets []DataCrunchSubnetSpec `json:"subnets,omitempty"`
}

// DataCrunchVPCSpec defines VPC configuration
type DataCrunchVPCSpec struct {
	// ID is the VPC ID to use. If not specified, a new VPC will be created
	// +optional
	ID string `json:"id,omitempty"`

	// CidrBlock is the CIDR block for the VPC
	// +optional
	CidrBlock string `json:"cidrBlock,omitempty"`

	// Tags to apply to the VPC
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
}

// DataCrunchSubnetSpec defines subnet configuration
type DataCrunchSubnetSpec struct {
	// ID is the subnet ID to use. If not specified, a new subnet will be created
	// +optional
	ID string `json:"id,omitempty"`

	// CidrBlock is the CIDR block for the subnet
	// +optional
	CidrBlock string `json:"cidrBlock,omitempty"`

	// AvailabilityZone is the availability zone for the subnet
	// +optional
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// IsPublic specifies whether the subnet is public
	// +optional
	IsPublic bool `json:"isPublic,omitempty"`

	// Tags to apply to the subnet
	// +optional
	Tags map[string]string `json:"tags,omitempty"`
}

// DataCrunchClusterStatus defines the observed state of DataCrunchCluster
type DataCrunchClusterStatus struct {
	// Ready denotes that the cluster (infrastructure) is ready.
	// +optional
	Ready bool `json:"ready"`

	// Conditions defines current service state of the DataCrunchCluster.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`

	// FailureDomains contains the failure domains that machines should be placed in.
	// +optional
	FailureDomains clusterv1.FailureDomains `json:"failureDomains,omitempty"`

	// Network contains information about the created network resources
	// +optional
	Network *DataCrunchNetworkStatus `json:"network,omitempty"`

	// LoadBalancer contains information about the control plane load balancer
	// +optional
	LoadBalancer *DataCrunchLoadBalancerStatus `json:"loadBalancer,omitempty"`
}

// DataCrunchNetworkStatus reports network status
type DataCrunchNetworkStatus struct {
	// VPC contains information about the VPC
	// +optional
	VPC *DataCrunchVPCStatus `json:"vpc,omitempty"`

	// Subnets contains information about the subnets
	// +optional
	Subnets []DataCrunchSubnetStatus `json:"subnets,omitempty"`
}

// DataCrunchVPCStatus reports VPC status
type DataCrunchVPCStatus struct {
	// ID is the VPC ID
	ID string `json:"id,omitempty"`

	// CidrBlock is the CIDR block of the VPC
	CidrBlock string `json:"cidrBlock,omitempty"`

	// State is the current state of the VPC
	State string `json:"state,omitempty"`
}

// DataCrunchSubnetStatus reports subnet status
type DataCrunchSubnetStatus struct {
	// ID is the subnet ID
	ID string `json:"id,omitempty"`

	// CidrBlock is the CIDR block of the subnet
	CidrBlock string `json:"cidrBlock,omitempty"`

	// AvailabilityZone is the availability zone of the subnet
	AvailabilityZone string `json:"availabilityZone,omitempty"`

	// State is the current state of the subnet
	State string `json:"state,omitempty"`
}

// DataCrunchLoadBalancerStatus reports load balancer status
type DataCrunchLoadBalancerStatus struct {
	// ID is the load balancer ID
	ID string `json:"id,omitempty"`

	// DNSName is the DNS name of the load balancer
	DNSName string `json:"dnsName,omitempty"`

	// State is the current state of the load balancer
	State string `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this DataCrunchCluster belongs"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Cluster infrastructure is ready for DataCrunch instances"
// +kubebuilder:printcolumn:name="VPC",type="string",JSONPath=".status.network.vpc.id",description="VPC ID"
// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".spec.controlPlaneEndpoint.host",description="API Endpoint",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time duration since creation of DataCrunchCluster"

// DataCrunchCluster is the Schema for the datacrunchclusters API
type DataCrunchCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataCrunchClusterSpec   `json:"spec,omitempty"`
	Status DataCrunchClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DataCrunchClusterList contains a list of DataCrunchCluster
type DataCrunchClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataCrunchCluster `json:"items"`
}

// GetConditions returns the observations of the operational state of the DataCrunchCluster resource.
func (c *DataCrunchCluster) GetConditions() clusterv1.Conditions {
	return c.Status.Conditions
}

// SetConditions sets the underlying service state of the DataCrunchCluster to the predicate provided.
func (c *DataCrunchCluster) SetConditions(conditions clusterv1.Conditions) {
	c.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&DataCrunchCluster{}, &DataCrunchClusterList{})
}
