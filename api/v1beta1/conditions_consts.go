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

import clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

// Condition types for DataCrunchCluster
const (
	// NetworkInfrastructureReadyCondition reports on the readiness of network infrastructure.
	NetworkInfrastructureReadyCondition clusterv1.ConditionType = "NetworkInfrastructureReady"

	// LoadBalancerReadyCondition reports on the readiness of the load balancer.
	LoadBalancerReadyCondition clusterv1.ConditionType = "LoadBalancerReady"
)

// Condition types for DataCrunchMachine
const (
	// InstanceReadyCondition reports on the readiness of the DataCrunch instance.
	InstanceReadyCondition clusterv1.ConditionType = "InstanceReady"
)

// Condition reasons for DataCrunchCluster
const (
	// DataCrunchClientFailedReason used when the DataCrunch client cannot be created.
	DataCrunchClientFailedReason = "DataCrunchClientFailed"

	// NetworkReconciliationFailedReason used when network reconciliation fails.
	NetworkReconciliationFailedReason = "NetworkReconciliationFailed"

	// LoadBalancerReconciliationFailedReason used when load balancer reconciliation fails.
	LoadBalancerReconciliationFailedReason = "LoadBalancerReconciliationFailed"
)

// Condition reasons for DataCrunchMachine
const (
	// WaitingForClusterInfrastructureReason used when machine is waiting for cluster infrastructure.
	WaitingForClusterInfrastructureReason = "WaitingForClusterInfrastructure"

	// WaitingForBootstrapDataReason used when machine is waiting for bootstrap data.
	WaitingForBootstrapDataReason = "WaitingForBootstrapData"

	// InstanceCreationFailedReason used when instance creation fails.
	InstanceCreationFailedReason = "InstanceCreationFailed"

	// InstanceNotReadyReason used when instance is not ready.
	InstanceNotReadyReason = "InstanceNotReady"

	// InstanceTerminatedReason used when instance is terminated.
	InstanceTerminatedReason = "InstanceTerminated"
)
