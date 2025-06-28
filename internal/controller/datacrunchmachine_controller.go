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

package controller

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	capierrors "sigs.k8s.io/cluster-api/errors"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	infrav1beta1 "github.com/rusik69/cluster-api-provider-datacrunch/api/v1beta1"
	"github.com/rusik69/cluster-api-provider-datacrunch/pkg/cloud"
	"github.com/rusik69/cluster-api-provider-datacrunch/pkg/cloud/datacrunch"
)

// DataCrunchMachineReconciler reconciles a DataCrunchMachine object
type DataCrunchMachineReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Log      logr.Logger

	WatchFilterValue string
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=datacrunchmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=datacrunchmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=datacrunchmachines/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DataCrunchMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := r.Log.WithValues("namespace", req.Namespace, "datacrunchMachine", req.Name)

	// Fetch the DataCrunchMachine instance
	dataCrunchMachine := &infrav1beta1.DataCrunchMachine{}
	err := r.Get(ctx, req.NamespacedName, dataCrunchMachine)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the Machine.
	machine, err := util.GetOwnerMachine(ctx, r.Client, dataCrunchMachine.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if machine == nil {
		log.Info("Machine Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}

	log = log.WithValues("machine", machine.Name)

	// Fetch the Cluster.
	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist")
		return reconcile.Result{}, nil
	}

	if annotations.IsPaused(cluster, dataCrunchMachine) {
		log.Info("DataCrunchMachine or linked Cluster is marked as paused. Won't reconcile")
		return reconcile.Result{}, nil
	}

	log = log.WithValues("cluster", cluster.Name)

	dataCrunchCluster := &infrav1beta1.DataCrunchCluster{}
	dataCrunchClusterName := client.ObjectKey{
		Namespace: dataCrunchMachine.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}
	if err := r.Get(ctx, dataCrunchClusterName, dataCrunchCluster); err != nil {
		log.Info("DataCrunchCluster is not available yet")
		return reconcile.Result{}, nil
	}

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(dataCrunchMachine, r.Client)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Always attempt to Patch the DataCrunchMachine object and status after each reconciliation.
	defer func() {
		if err := patchHelper.Patch(ctx, dataCrunchMachine); err != nil {
			log.Error(err, "failed to patch DataCrunchMachine")
			if reterr == nil {
				reterr = err
			}
		}
	}()

	// Handle deleted machines
	if !dataCrunchMachine.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, log, machine, dataCrunchMachine, cluster, dataCrunchCluster)
	}

	// Handle non-deleted machines
	return r.reconcileNormal(ctx, log, machine, dataCrunchMachine, cluster, dataCrunchCluster)
}

func (r *DataCrunchMachineReconciler) reconcileNormal(ctx context.Context, log logr.Logger, machine *clusterv1.Machine, dataCrunchMachine *infrav1beta1.DataCrunchMachine, cluster *clusterv1.Cluster, dataCrunchCluster *infrav1beta1.DataCrunchCluster) (reconcile.Result, error) {
	log.Info("Reconciling DataCrunchMachine")

	// If the DataCrunchMachine is in an error state, return early.
	if dataCrunchMachine.Status.FailureReason != nil || dataCrunchMachine.Status.FailureMessage != nil {
		log.Info("Error state detected, skipping reconciliation")
		return reconcile.Result{}, nil
	}

	// If the DataCrunchMachine doesn't have our finalizer, add it.
	if !controllerutil.ContainsFinalizer(dataCrunchMachine, infrav1beta1.MachineFinalizer) {
		controllerutil.AddFinalizer(dataCrunchMachine, infrav1beta1.MachineFinalizer)
		return reconcile.Result{}, nil
	}

	if !cluster.Status.InfrastructureReady {
		log.Info("Cluster infrastructure is not ready yet")
		conditions.MarkFalse(dataCrunchMachine, infrav1beta1.InstanceReadyCondition, infrav1beta1.WaitingForClusterInfrastructureReason, clusterv1.ConditionSeverityInfo, "")
		return reconcile.Result{}, nil
	}

	// Make sure bootstrap data is available and populated.
	if machine.Spec.Bootstrap.DataSecretName == nil {
		log.Info("Bootstrap data secret reference is not yet available")
		conditions.MarkFalse(dataCrunchMachine, infrav1beta1.InstanceReadyCondition, infrav1beta1.WaitingForBootstrapDataReason, clusterv1.ConditionSeverityInfo, "")
		return reconcile.Result{}, nil
	}

	// Create DataCrunch client
	dataCrunchClient, err := r.createDataCrunchClient(ctx, cluster)
	if err != nil {
		log.Error(err, "failed to create DataCrunch client")
		conditions.MarkFalse(dataCrunchMachine, infrav1beta1.InstanceReadyCondition, infrav1beta1.DataCrunchClientFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return reconcile.Result{}, err
	}

	// Try to find existing instance
	instance, err := r.findInstance(ctx, dataCrunchClient, dataCrunchMachine)
	if err != nil {
		log.Error(err, "failed to query for existing instance")
		return reconcile.Result{}, err
	}

	if instance == nil {
		// Instance doesn't exist, so create it
		instance, err = r.createInstance(ctx, log, dataCrunchClient, machine, dataCrunchMachine, cluster, dataCrunchCluster)
		if err != nil {
			log.Error(err, "failed to create instance")
			conditions.MarkFalse(dataCrunchMachine, infrav1beta1.InstanceReadyCondition, infrav1beta1.InstanceCreationFailedReason, clusterv1.ConditionSeverityError, err.Error())
			return reconcile.Result{}, err
		}

		log.Info("Created new DataCrunch instance", "instanceId", instance.ID)
		conditions.MarkFalse(dataCrunchMachine, infrav1beta1.InstanceReadyCondition, infrav1beta1.InstanceNotReadyReason, clusterv1.ConditionSeverityInfo, "")
		r.Recorder.Eventf(dataCrunchMachine, corev1.EventTypeNormal, "InstanceCreated", "Created new DataCrunch instance %s", instance.ID)
	}

	// Set the provider ID to identify the instance
	if dataCrunchMachine.Spec.ProviderID == nil {
		providerID := fmt.Sprintf("datacrunch://%s", instance.ID)
		dataCrunchMachine.Spec.ProviderID = &providerID
	}

	// Update machine status based on instance state
	dataCrunchMachine.Status.InstanceState = (*infrav1beta1.InstanceState)(&instance.State)

	switch instance.State {
	case "running":
		log.Info("DataCrunch instance is running", "instanceId", instance.ID)
		dataCrunchMachine.Status.Ready = true
		conditions.MarkTrue(dataCrunchMachine, infrav1beta1.InstanceReadyCondition)

		// Set machine addresses
		dataCrunchMachine.Status.Addresses = []clusterv1.MachineAddress{
			{
				Type:    clusterv1.MachineHostName,
				Address: instance.Name,
			},
		}

		if instance.PrivateIP != "" {
			dataCrunchMachine.Status.Addresses = append(dataCrunchMachine.Status.Addresses, clusterv1.MachineAddress{
				Type:    clusterv1.MachineInternalIP,
				Address: instance.PrivateIP,
			})
		}

		if instance.PublicIP != "" {
			dataCrunchMachine.Status.Addresses = append(dataCrunchMachine.Status.Addresses, clusterv1.MachineAddress{
				Type:    clusterv1.MachineExternalIP,
				Address: instance.PublicIP,
			})
		}

	case "pending":
		log.Info("DataCrunch instance is pending", "instanceId", instance.ID)
		conditions.MarkFalse(dataCrunchMachine, infrav1beta1.InstanceReadyCondition, infrav1beta1.InstanceNotReadyReason, clusterv1.ConditionSeverityInfo, "Instance is pending")
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil

	case "stopped":
		log.Info("DataCrunch instance is stopped, starting it", "instanceId", instance.ID)
		if err := dataCrunchClient.StartInstance(ctx, instance.ID); err != nil {
			log.Error(err, "failed to start instance")
			return reconcile.Result{RequeueAfter: 30 * time.Second}, err
		}
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil

	case "terminated":
		log.Info("DataCrunch instance is terminated")
		failureReason := capierrors.UpdateMachineError
		failureMessage := "Instance was terminated"
		dataCrunchMachine.Status.FailureReason = &failureReason
		dataCrunchMachine.Status.FailureMessage = &failureMessage
		conditions.MarkFalse(dataCrunchMachine, infrav1beta1.InstanceReadyCondition, infrav1beta1.InstanceTerminatedReason, clusterv1.ConditionSeverityError, "Instance was terminated")
		return reconcile.Result{}, nil

	default:
		log.Info("DataCrunch instance is in unknown state", "state", instance.State, "instanceId", instance.ID)
		conditions.MarkFalse(dataCrunchMachine, infrav1beta1.InstanceReadyCondition, infrav1beta1.InstanceNotReadyReason, clusterv1.ConditionSeverityWarning, fmt.Sprintf("Instance is in unknown state: %s", instance.State))
		return reconcile.Result{RequeueAfter: 30 * time.Second}, nil
	}

	log.Info("Successfully reconciled DataCrunchMachine")
	return reconcile.Result{}, nil
}

func (r *DataCrunchMachineReconciler) reconcileDelete(ctx context.Context, log logr.Logger, machine *clusterv1.Machine, dataCrunchMachine *infrav1beta1.DataCrunchMachine, cluster *clusterv1.Cluster, dataCrunchCluster *infrav1beta1.DataCrunchCluster) (reconcile.Result, error) {
	log.Info("Reconciling DataCrunchMachine delete")

	// Create DataCrunch client
	dataCrunchClient, err := r.createDataCrunchClient(ctx, cluster)
	if err != nil {
		log.Error(err, "failed to create DataCrunch client during deletion")
		// Continue with deletion even if we can't create the client
	}

	// Try to find and delete the instance
	if dataCrunchClient != nil {
		instance, err := r.findInstance(ctx, dataCrunchClient, dataCrunchMachine)
		if err != nil {
			log.Error(err, "failed to find instance during deletion")
		} else if instance != nil {
			log.Info("Deleting DataCrunch instance", "instanceId", instance.ID)
			if err := dataCrunchClient.DeleteInstance(ctx, instance.ID); err != nil {
				log.Error(err, "failed to delete instance")
				return reconcile.Result{RequeueAfter: 30 * time.Second}, err
			}
			r.Recorder.Eventf(dataCrunchMachine, corev1.EventTypeNormal, "InstanceDeleted", "Deleted DataCrunch instance %s", instance.ID)
		}
	}

	// Remove our finalizer from the list and update it
	controllerutil.RemoveFinalizer(dataCrunchMachine, infrav1beta1.MachineFinalizer)

	log.Info("Successfully reconciled DataCrunchMachine delete")
	return reconcile.Result{}, nil
}

func (r *DataCrunchMachineReconciler) findInstance(ctx context.Context, dataCrunchClient cloud.Client, dataCrunchMachine *infrav1beta1.DataCrunchMachine) (*cloud.Instance, error) {
	if dataCrunchMachine.Spec.ProviderID != nil {
		// Extract instance ID from provider ID (format: datacrunch://instance-id)
		instanceID := ""
		if len(*dataCrunchMachine.Spec.ProviderID) > 13 { // len("datacrunch://") = 13
			instanceID = (*dataCrunchMachine.Spec.ProviderID)[13:]
		}

		if instanceID != "" {
			instance, err := dataCrunchClient.GetInstance(ctx, instanceID)
			if err != nil {
				// Instance not found is not an error during deletion
				if err.Error() == fmt.Sprintf("instance not found: %s", instanceID) {
					return nil, nil
				}
				return nil, err
			}
			return instance, nil
		}
	}

	// If no provider ID, we can't find the instance
	return nil, nil
}

func (r *DataCrunchMachineReconciler) createInstance(ctx context.Context, log logr.Logger, dataCrunchClient cloud.Client, machine *clusterv1.Machine, dataCrunchMachine *infrav1beta1.DataCrunchMachine, cluster *clusterv1.Cluster, dataCrunchCluster *infrav1beta1.DataCrunchCluster) (*cloud.Instance, error) {
	// Get bootstrap data
	userData, err := r.getBootstrapData(ctx, machine)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get bootstrap data")
	}

	// Prepare instance specification
	instanceSpec := &cloud.InstanceSpec{
		Name:         dataCrunchMachine.Name,
		InstanceType: dataCrunchMachine.Spec.InstanceType,
		ImageID:      dataCrunchMachine.Spec.Image,
		SSHKeyName:   dataCrunchMachine.Spec.SSHKeyName,
		UserData:     userData,
		Metadata:     dataCrunchMachine.Spec.AdditionalMetadata,
		Tags:         dataCrunchMachine.Spec.AdditionalTags,
		PublicIP:     dataCrunchMachine.Spec.PublicIP != nil && *dataCrunchMachine.Spec.PublicIP,
	}

	// Set default image if not specified
	if instanceSpec.ImageID == "" {
		instanceSpec.ImageID = "ubuntu-22.04-cuda-12.1"
	}

	// Add cluster and machine labels to tags
	if instanceSpec.Tags == nil {
		instanceSpec.Tags = make(map[string]string)
	}
	instanceSpec.Tags["cluster.x-k8s.io/cluster-name"] = cluster.Name
	instanceSpec.Tags["cluster.x-k8s.io/machine-name"] = machine.Name

	// Create the instance
	instance, err := dataCrunchClient.CreateInstance(ctx, instanceSpec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create DataCrunch instance")
	}

	return instance, nil
}

func (r *DataCrunchMachineReconciler) getBootstrapData(ctx context.Context, machine *clusterv1.Machine) (string, error) {
	if machine.Spec.Bootstrap.DataSecretName == nil {
		return "", errors.New("error retrieving bootstrap data: linked Machine's bootstrap.dataSecretName is nil")
	}

	secret := &corev1.Secret{}
	key := client.ObjectKey{Namespace: machine.Namespace, Name: *machine.Spec.Bootstrap.DataSecretName}
	if err := r.Get(ctx, key, secret); err != nil {
		return "", errors.Wrapf(err, "failed to retrieve bootstrap data secret for DataCrunchMachine %s/%s", machine.Namespace, machine.Name)
	}

	value, ok := secret.Data["value"]
	if !ok {
		return "", errors.New("error retrieving bootstrap data: secret value key is missing")
	}

	return base64.StdEncoding.EncodeToString(value), nil
}

func (r *DataCrunchMachineReconciler) createDataCrunchClient(ctx context.Context, cluster *clusterv1.Cluster) (cloud.Client, error) {
	// In a real implementation, you would get credentials from a secret
	// For now, we'll use placeholder credentials
	clientID := "your-datacrunch-client-id"
	clientSecret := "your-datacrunch-client-secret"

	return datacrunch.NewClient(clientID, clientSecret), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DataCrunchMachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1beta1.DataCrunchMachine{}).
		WithOptions(options).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(log, r.WatchFilterValue)).
		Complete(r)
}
