// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/controllers/remote"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrav1 "github.com/liquidmetal-dev/cluster-api-provider-microvm/api/v1alpha1"
	"github.com/liquidmetal-dev/cluster-api-provider-microvm/internal/defaults"
	"github.com/liquidmetal-dev/cluster-api-provider-microvm/internal/scope"
)

const (
	requeuePeriod = 30 * time.Second
)

// MicrovmClusterReconciler reconciles a MicrovmCluster object.
type MicrovmClusterReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	Recorder         record.EventRecorder
	WatchFilterValue string

	RemoteClientGetter remote.ClusterClientGetter
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=microvmclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=microvmclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=microvmclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *MicrovmClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	mvmCluster := &infrav1.MicrovmCluster{}

	err := r.Get(ctx, req.NamespacedName, mvmCluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		log.Error(err, "error getting microvmcluster", "id", req.NamespacedName)

		return ctrl.Result{}, fmt.Errorf("error getting microvmcluster: %w", err)
	}

	cluster, err := util.GetOwnerCluster(ctx, r.Client, mvmCluster.ObjectMeta)
	if err != nil {
		log.Error(err, "getting owning cluster")

		return ctrl.Result{}, fmt.Errorf("error getting owning cluster: %w", err)
	}

	if cluster == nil {
		log.Info("Cluster Controller has not yet set OwnerRef")

		return ctrl.Result{}, nil
	}

	if annotations.IsPaused(cluster, mvmCluster) {
		log.Info("MicrovmCluster or linked Cluster is marked as paused. Won't reconcile")

		return ctrl.Result{}, nil
	}

	scope, err := scope.NewClusterScope(cluster,
		mvmCluster,
		r.Client,
		scope.WithClusterLogger(log.WithValues("microvmcluster", req.NamespacedName)))
	if err != nil {
		log.Error(err, "creating cluster scope")

		return ctrl.Result{}, err
	}

	defer func() {
		if patchErr := scope.Patch(); patchErr != nil {
			log.Error(patchErr, "failed to patch microvm cluster")
		}
	}()

	if !mvmCluster.DeletionTimestamp.IsZero() {
		log.Info("Deleting cluster")

		return r.reconcileDelete(ctx, scope)
	}

	return r.reconcileNormal(ctx, scope)
}

func (r *MicrovmClusterReconciler) reconcileDelete(
	_ context.Context,
	clusterScope *scope.ClusterScope,
) (reconcile.Result, error) {
	clusterScope.Info("Reconciling MicrovmCluster delete")

	// We currently do not do any Cluster creation so there is nothing to delete.

	return reconcile.Result{}, nil
}

func (r *MicrovmClusterReconciler) reconcileNormal(
	ctx context.Context,
	cScope *scope.ClusterScope,
) (reconcile.Result, error) {
	cScope.Info("Reconciling MicrovmCluster")

	if cScope.Cluster.Spec.ControlPlaneEndpoint.IsZero() && cScope.MvmCluster.Spec.ControlPlaneEndpoint.IsZero() {
		return reconcile.Result{}, errControlplaneEndpointRequired
	}

	cScope.MvmCluster.Status.Ready = true

	if err := r.setFailureDomains(cScope); err != nil {
		return reconcile.Result{}, fmt.Errorf("setting failuredomains: %w", err)
	}

	available := r.isAPIServerAvailable(ctx, cScope)
	if !available {
		conditions.MarkFalse(
			cScope.MvmCluster,
			infrav1.LoadBalancerAvailableCondition,
			infrav1.LoadBalancerNotAvailableReason,
			clusterv1.ConditionSeverityInfo,
			"control plane load balancer isn't available",
		)

		return reconcile.Result{RequeueAfter: requeuePeriod}, nil
	}

	conditions.MarkTrue(cScope.MvmCluster, infrav1.LoadBalancerAvailableCondition)

	return reconcile.Result{}, nil
}

func (r *MicrovmClusterReconciler) isAPIServerAvailable(ctx context.Context, clusterScope *scope.ClusterScope) bool {
	clusterScope.
		V(defaults.LogLevelDebug).
		Info("checking if api server is available", "cluster", clusterScope.ClusterName())

	clusterKey := client.ObjectKey{
		Name:      clusterScope.Cluster.Name,
		Namespace: clusterScope.Cluster.Namespace,
	}

	remoteClient, err := r.RemoteClientGetter(ctx, clusterScope.ClusterName(), r.Client, clusterKey)
	if err != nil {
		clusterScope.Error(err, "creating remote cluster client")

		return false
	}

	nodes := &corev1.NodeList{}
	if err = remoteClient.List(ctx, nodes); err != nil {
		return false
	}

	if len(nodes.Items) == 0 {
		return false
	}

	clusterScope.Info("api server is available", "cluster", clusterScope.ClusterName())

	return true
}

func (r *MicrovmClusterReconciler) setFailureDomains(clusterScope *scope.ClusterScope) error {
	placement := clusterScope.Placement()

	if !placement.IsSet() {
		return errNoPlacement
	}

	if placement.StaticPool != nil {
		clusterScope.Info("using static pool placement")

		failureDomains := clusterv1.FailureDomains{}

		for _, host := range placement.StaticPool.Hosts {
			clusterScope.
				V(defaults.LogLevelTrace).
				Info(
					"adding failure domain",
					"endpoint", host.Endpoint,
					"name", host.Name,
					"controlplane", host.ControlPlaneAllowed,
				)

			failureDomains[host.Endpoint] = clusterv1.FailureDomainSpec{
				ControlPlane: host.ControlPlaneAllowed,
			}
		}

		clusterScope.MvmCluster.Status.FailureDomains = failureDomains
	}
	// NOTE: additional placement methods can be added the future

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MicrovmClusterReconciler) SetupWithManager(
	ctx context.Context,
	mgr ctrl.Manager,
	options controller.Options,
) error {
	log := ctrl.LoggerFrom(ctx)

	if r.RemoteClientGetter == nil {
		r.RemoteClientGetter = remote.NewClusterClient
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.MicrovmCluster{}).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(log, r.WatchFilterValue)).
		WithEventFilter(predicates.ResourceIsNotExternallyManaged(log)).
		Watches(
			&source.Kind{Type: &clusterv1.Cluster{}},
			handler.EnqueueRequestsFromMapFunc(
				util.ClusterToInfrastructureMapFunc(ctx,
					infrav1.GroupVersion.WithKind("MicrovmCluster"),
					r.Client,
					&clusterv1.Cluster{},
				),
			),
			builder.WithPredicates(
				predicates.ClusterUnpaused(log),
			),
		)

	if err := builder.Complete(r); err != nil {
		return fmt.Errorf("creating microvm cluster controller: %w", err)
	}

	return nil
}
