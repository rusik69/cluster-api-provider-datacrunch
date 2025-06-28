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
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
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

// DataCrunchClusterReconciler reconciles a DataCrunchCluster object
type DataCrunchClusterReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Log      logr.Logger

	WatchFilterValue string
}

//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=datacrunchclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=datacrunchclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=datacrunchclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DataCrunchClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := r.Log.WithValues("namespace", req.Namespace, "datacrunchCluster", req.Name)

	// Fetch the DataCrunchCluster instance
	dataCrunchCluster := &infrav1beta1.DataCrunchCluster{}
	err := r.Get(ctx, req.NamespacedName, dataCrunchCluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, dataCrunchCluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		log.Info("Cluster Controller has not yet set OwnerRef")
		return reconcile.Result{}, nil
	}

	log = log.WithValues("cluster", cluster.Name)

	if annotations.IsPaused(cluster, dataCrunchCluster) {
		log.Info("DataCrunchCluster or linked Cluster is marked as paused. Won't reconcile")
		return reconcile.Result{}, nil
	}

	// Initialize the patch helper
	patchHelper, err := patch.NewHelper(dataCrunchCluster, r.Client)
	if err != nil {
		return reconcile.Result{}, err
	}

	// Always attempt to Patch the DataCrunchCluster object and status after each reconciliation.
	defer func() {
		if err := patchHelper.Patch(ctx, dataCrunchCluster); err != nil {
			log.Error(err, "failed to patch DataCrunchCluster")
			if reterr == nil {
				reterr = err
			}
		}
	}()

	// Handle deleted clusters
	if !dataCrunchCluster.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, log, cluster, dataCrunchCluster)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, log, cluster, dataCrunchCluster)
}

func (r *DataCrunchClusterReconciler) reconcileNormal(ctx context.Context, log logr.Logger, cluster *clusterv1.Cluster, dataCrunchCluster *infrav1beta1.DataCrunchCluster) (reconcile.Result, error) {
	log.Info("Reconciling DataCrunchCluster")

	// If the DataCrunchCluster doesn't have our finalizer, add it.
	if !controllerutil.ContainsFinalizer(dataCrunchCluster, infrav1beta1.ClusterFinalizer) {
		controllerutil.AddFinalizer(dataCrunchCluster, infrav1beta1.ClusterFinalizer)
		return reconcile.Result{}, nil
	}

	// Create DataCrunch client
	dataCrunchClient, err := r.createDataCrunchClient(ctx, cluster)
	if err != nil {
		log.Error(err, "failed to create DataCrunch client")
		conditions.MarkFalse(dataCrunchCluster, infrav1beta1.NetworkInfrastructureReadyCondition, infrav1beta1.DataCrunchClientFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return reconcile.Result{}, err
	}

	// Reconcile network infrastructure
	if err := r.reconcileNetwork(ctx, log, dataCrunchClient, dataCrunchCluster); err != nil {
		log.Error(err, "failed to reconcile network infrastructure")
		conditions.MarkFalse(dataCrunchCluster, infrav1beta1.NetworkInfrastructureReadyCondition, infrav1beta1.NetworkReconciliationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return reconcile.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Reconcile load balancer if needed
	if err := r.reconcileLoadBalancer(ctx, log, dataCrunchClient, cluster, dataCrunchCluster); err != nil {
		log.Error(err, "failed to reconcile load balancer")
		conditions.MarkFalse(dataCrunchCluster, infrav1beta1.LoadBalancerReadyCondition, infrav1beta1.LoadBalancerReconciliationFailedReason, clusterv1.ConditionSeverityError, err.Error())
		return reconcile.Result{RequeueAfter: 30 * time.Second}, err
	}

	// Mark the cluster as ready
	dataCrunchCluster.Status.Ready = true
	conditions.MarkTrue(dataCrunchCluster, infrav1beta1.NetworkInfrastructureReadyCondition)

	log.Info("Successfully reconciled DataCrunchCluster")
	return reconcile.Result{}, nil
}

func (r *DataCrunchClusterReconciler) reconcileDelete(ctx context.Context, log logr.Logger, cluster *clusterv1.Cluster, dataCrunchCluster *infrav1beta1.DataCrunchCluster) (reconcile.Result, error) {
	log.Info("Reconciling DataCrunchCluster delete")

	// Create DataCrunch client
	dataCrunchClient, err := r.createDataCrunchClient(ctx, cluster)
	if err != nil {
		log.Error(err, "failed to create DataCrunch client during deletion")
		// Continue with deletion even if we can't create the client
	}

	// Delete load balancer if it exists
	if dataCrunchClient != nil && dataCrunchCluster.Status.LoadBalancer != nil {
		if err := dataCrunchClient.DeleteLoadBalancer(ctx, dataCrunchCluster.Status.LoadBalancer.ID); err != nil {
			log.Error(err, "failed to delete load balancer")
			// Don't return error, continue with cleanup
		}
	}

	// Clean up network resources would go here
	// For now, we'll just remove the finalizer

	// Remove our finalizer from the list and update it
	controllerutil.RemoveFinalizer(dataCrunchCluster, infrav1beta1.ClusterFinalizer)

	log.Info("Successfully reconciled DataCrunchCluster delete")
	return reconcile.Result{}, nil
}

func (r *DataCrunchClusterReconciler) reconcileNetwork(ctx context.Context, log logr.Logger, dataCrunchClient cloud.Client, dataCrunchCluster *infrav1beta1.DataCrunchCluster) error {
	// For now, we'll assume the network infrastructure is ready
	// In a real implementation, you would create VPCs, subnets, security groups, etc.

	log.Info("Network infrastructure reconciliation completed")

	// Initialize network status if not exists
	if dataCrunchCluster.Status.Network == nil {
		dataCrunchCluster.Status.Network = &infrav1beta1.DataCrunchNetworkStatus{}
	}

	// Set default failure domains
	if dataCrunchCluster.Status.FailureDomains == nil {
		dataCrunchCluster.Status.FailureDomains = clusterv1.FailureDomains{
			"default": clusterv1.FailureDomainSpec{
				ControlPlane: true,
			},
		}
	}

	return nil
}

func (r *DataCrunchClusterReconciler) reconcileLoadBalancer(ctx context.Context, log logr.Logger, dataCrunchClient cloud.Client, cluster *clusterv1.Cluster, dataCrunchCluster *infrav1beta1.DataCrunchCluster) error {
	// Check if control plane endpoint is already set
	if !dataCrunchCluster.Spec.ControlPlaneEndpoint.IsZero() {
		log.Info("Control plane endpoint already set", "endpoint", dataCrunchCluster.Spec.ControlPlaneEndpoint)
		return nil
	}

	// For simplicity, we'll set a placeholder endpoint
	// In a real implementation, you would create a load balancer and get its endpoint
	dataCrunchCluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
		Host: "cluster-" + cluster.Name + ".datacrunch.local",
		Port: 6443,
	}

	log.Info("Set control plane endpoint", "endpoint", dataCrunchCluster.Spec.ControlPlaneEndpoint)
	return nil
}

func (r *DataCrunchClusterReconciler) createDataCrunchClient(ctx context.Context, cluster *clusterv1.Cluster) (cloud.Client, error) {
	// In a real implementation, you would get credentials from a secret
	// For now, we'll use placeholder credentials
	clientID := "your-datacrunch-client-id"
	clientSecret := "your-datacrunch-client-secret"

	return datacrunch.NewClient(clientID, clientSecret), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DataCrunchClusterReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)

	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1beta1.DataCrunchCluster{}).
		WithOptions(options).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(log, r.WatchFilterValue)).
		Complete(r)
}
