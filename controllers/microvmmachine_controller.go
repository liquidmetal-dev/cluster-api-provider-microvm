// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0.

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	flintlocktypes "github.com/weaveworks/flintlock/api/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/annotations"
	"sigs.k8s.io/cluster-api/util/collections"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrav1 "github.com/weaveworks/cluster-api-provider-microvm/api/v1alpha1"
	"github.com/weaveworks/cluster-api-provider-microvm/internal/defaults"
	"github.com/weaveworks/cluster-api-provider-microvm/internal/scope"
	"github.com/weaveworks/cluster-api-provider-microvm/internal/services/microvm"
)

// MicrovmMachineReconciler reconciles a MicrovmMachine object.
type MicrovmMachineReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	Recorder         record.EventRecorder
	WatchFilterValue string

	MvmClientFunc microvm.ClientFactoryFunc
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=microvmmachines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=microvmmachines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=microvmmachines/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines;machines/status,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *MicrovmMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	mvmMachine := &infrav1.MicrovmMachine{}
	if err := r.Get(ctx, req.NamespacedName, mvmMachine); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		log.Error(err, "error getting microvmmachine", "id", req.NamespacedName)

		return ctrl.Result{}, fmt.Errorf("unable to reconcile: %w", err)
	}

	machine, err := util.GetOwnerMachine(ctx, r.Client, mvmMachine.ObjectMeta)
	if err != nil {
		log.Error(err, "getting owning machine")

		return ctrl.Result{}, fmt.Errorf("unable to get machine owner: %w", err)
	}

	if machine == nil {
		log.Info("Machine controller has not set OwnerRef")

		return ctrl.Result{}, nil
	}

	log = log.WithValues("machine", machine.Name)

	cluster, err := util.GetClusterFromMetadata(ctx, r.Client, machine.ObjectMeta)
	if err != nil {
		log.Info("Machine is missing cluster label or cluster does not exist")

		return ctrl.Result{}, nil //nolint:nilerr // We ignore it intentionally.
	}

	if annotations.IsPaused(cluster, mvmMachine) {
		log.Info("MicrovmMachine or linked Cluster is marked as paused. Won't reconcile")

		return ctrl.Result{}, nil
	}

	log = log.WithValues("cluster", cluster.Name)

	mvmCluster := &infrav1.MicrovmCluster{}
	mvmClusterName := client.ObjectKey{
		Namespace: cluster.Spec.InfrastructureRef.Namespace,
		Name:      cluster.Spec.InfrastructureRef.Name,
	}

	if getErr := r.Client.Get(ctx, mvmClusterName, mvmCluster); getErr != nil {
		if apierrors.IsNotFound(getErr) {
			log.Info("MicrovmCluster is not ready yet")

			return ctrl.Result{}, nil
		}

		log.Error(getErr, "error getting microvmcluster", "id", mvmClusterName)

		return ctrl.Result{}, fmt.Errorf("error getting microvmcluster: %w", getErr)
	}

	machineScope, err := scope.NewMachineScope(scope.MachineScopeParams{
		Cluster:        cluster,
		MicroVMCluster: mvmCluster,
		Machine:        machine,
		MicroVMMachine: mvmMachine,
		Client:         r.Client,
	})
	if err != nil {
		log.Error(err, "failed to create machine scope")

		return ctrl.Result{}, fmt.Errorf("failed to create machine scope: %w", err)
	}

	defer func() {
		if patchErr := machineScope.Patch(); patchErr != nil {
			log.Error(patchErr, "failed to patch microvm machine")
		}
	}()

	if !mvmMachine.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("Deleting machine")

		return r.reconcileDelete(ctx, machineScope)
	}

	return r.reconcileNormal(ctx, machineScope)
}

func (r *MicrovmMachineReconciler) reconcileDelete(
	ctx context.Context,
	machineScope *scope.MachineScope,
) (reconcile.Result, error) {
	machineScope.Info("Reconciling MicrovmMachine delete")

	mvmSvc, err := r.getMicrovmService(machineScope)
	if err != nil {
		machineScope.Error(err, "failed to get microvm service")

		return ctrl.Result{}, nil
	}

	microvm, err := mvmSvc.Get(ctx)
	if err != nil && !isSpecNotFound(err) {
		machineScope.Error(err, "failed getting microvm")

		return ctrl.Result{}, fmt.Errorf("failed getting microvm: %w", err)
	}

	if microvm != nil {
		machineScope.Info("deleting microvm")

		// Mark the machine as no longer ready before we delete.
		machineScope.SetNotReady(infrav1.MicrovmDeletingReason, clusterv1.ConditionSeverityInfo, "")

		if err := machineScope.Patch(); err != nil {
			machineScope.Error(err, "failed to patch object")

			return ctrl.Result{}, err
		}

		if microvm.Status.State != flintlocktypes.MicroVMStatus_DELETING {
			if _, err := mvmSvc.Delete(ctx); err != nil {
				machineScope.SetNotReady(infrav1.MicrovmDeleteFailedReason, clusterv1.ConditionSeverityError, "")

				return ctrl.Result{}, err
			}
		}

		return ctrl.Result{RequeueAfter: requeuePeriod}, nil
	}

	// By this point Flintlock has no record of the MvM, so we are good to clear
	// the finalizer
	controllerutil.RemoveFinalizer(machineScope.MvmMachine, infrav1.MachineFinalizer)

	machineScope.Info("microvm deleted")

	return ctrl.Result{}, nil
}

func (r *MicrovmMachineReconciler) reconcileNormal(
	ctx context.Context,
	machineScope *scope.MachineScope,
) (reconcile.Result, error) {
	machineScope.Info("Reconciling MicrovmMachine")

	if !machineScope.Cluster.Status.InfrastructureReady {
		machineScope.Info("Cluster infrastructure is not ready")
		conditions.MarkFalse(
			machineScope.MvmMachine, infrav1.MicrovmReadyCondition,
			infrav1.WaitingForClusterInfraReason, clusterv1.ConditionSeverityInfo,
			"",
		)

		return ctrl.Result{}, nil
	}

	if machineScope.Machine.Spec.Bootstrap.DataSecretName == nil {
		machineScope.Info("Bootstrap secret is not ready")
		conditions.MarkFalse(
			machineScope.MvmMachine, infrav1.MicrovmReadyCondition,
			infrav1.WaitingForBootstrapDataReason, clusterv1.ConditionSeverityInfo,
			"",
		)

		return ctrl.Result{}, nil
	}

	machineScope.
		V(defaults.LogLevelDebug).
		Info(
			"Bootstrap secret is ready", "machine", machineScope.MvmMachine.Name, "secret",
			machineScope.Machine.Spec.Bootstrap.DataSecretName)

	mvmSvc, err := r.getMicrovmService(machineScope)
	if err != nil {
		machineScope.Error(err, "failed to get microvm machine")

		return ctrl.Result{}, err
	}

	var microvm *flintlocktypes.MicroVM

	if machineScope.MvmMachine.Spec.ProviderID != nil {
		var err error
		microvm, err = mvmSvc.Get(ctx)
		if err != nil && !isSpecNotFound(err) {
			machineScope.Error(err, "failed checking if microvm exists")

			return ctrl.Result{}, err
		}
	}

	controllerutil.AddFinalizer(machineScope.MvmMachine, infrav1.MachineFinalizer)

	if err := machineScope.Patch(); err != nil {
		machineScope.Error(err, "unable to patch microvm machine")

		return ctrl.Result{}, err
	}

	if microvm == nil {
		machineScope.Info("creating microvm")

		var createErr error
		microvm, createErr = mvmSvc.Create(ctx)
		if createErr != nil {
			return ctrl.Result{}, createErr
		}
	}

	machineScope.SetProviderID(microvm.Spec.Uid)
	if err := machineScope.Patch(); err != nil {
		machineScope.Error(err, "unable to patch microvm machine")

		return ctrl.Result{}, err
	}

	switch microvm.Status.State {
	case flintlocktypes.MicroVMStatus_FAILED:
		// TODO: we need a failure reason from flintlock: Flintlock #299
		machineScope.MvmMachine.Status.VMState = &infrav1.VMStateFailed
		machineScope.SetNotReady(infrav1.MicrovmProvisionFailedReason,
			clusterv1.ConditionSeverityError,
			errMicrovmFailed.Error(),
		)

		return ctrl.Result{}, errMicrovmFailed
	case flintlocktypes.MicroVMStatus_PENDING:
		machineScope.MvmMachine.Status.VMState = &infrav1.VMStatePending
		machineScope.SetNotReady(infrav1.MicrovmPendingReason, clusterv1.ConditionSeverityInfo, "")

		return ctrl.Result{RequeueAfter: requeuePeriod}, nil
	case flintlocktypes.MicroVMStatus_CREATED:
		machineScope.MvmMachine.Status.VMState = &infrav1.VMStateRunning
		machineScope.V(defaults.LogLevelDebug).Info("microvm is in created state")
	case flintlocktypes.MicroVMStatus_DELETING:
		machineScope.V(defaults.LogLevelDebug).Info("microvm is deleting")

		return ctrl.Result{RequeueAfter: requeuePeriod}, nil
	default:
		machineScope.MvmMachine.Status.VMState = &infrav1.VMStateUnknown
		machineScope.SetNotReady(
			infrav1.MicrovmUnknownStateReason,
			clusterv1.ConditionSeverityError,
			errMicrovmUnknownState.Error(),
		)

		return ctrl.Result{RequeueAfter: requeuePeriod}, errMicrovmUnknownState
	}

	machineScope.Info("microvm created", "name", machineScope.Name(), "UID", machineScope.UID())

	machineScope.SetReady()

	return reconcile.Result{}, nil
}

func (r *MicrovmMachineReconciler) getMicrovmService(machineScope *scope.MachineScope) (*microvm.Service, error) {
	if r.MvmClientFunc == nil {
		return nil, errClientFactoryFuncRequired
	}

	addr, err := machineScope.MicrovmServiceAddress()
	if err != nil {
		return nil, err
	}

	client, err := r.MvmClientFunc(addr)
	if err != nil {
		return nil, fmt.Errorf("creating microvm client: %w", err)
	}

	return microvm.New(machineScope, client), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MicrovmMachineReconciler) SetupWithManager(
	ctx context.Context,
	mgr ctrl.Manager,
	options controller.Options,
) error {
	log := ctrl.LoggerFrom(ctx)

	clusterToObjectFunc, err := util.ClusterToObjectsMapper(
		r.Client,
		&infrav1.MicrovmMachineList{},
		mgr.GetScheme(),
	)
	if err != nil {
		return fmt.Errorf("failed to create mapper for Cluster to MicrovmMachines: %w", err)
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.MicrovmMachine{}).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(log, r.WatchFilterValue)).
		WithEventFilter(predicates.ResourceIsNotExternallyManaged(log)).
		Watches(
			&source.Kind{Type: &clusterv1.Machine{}},
			handler.EnqueueRequestsFromMapFunc(
				util.MachineToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("MicrovmMachine")),
			),
		).
		Watches(
			&source.Kind{Type: &infrav1.MicrovmCluster{}},
			handler.EnqueueRequestsFromMapFunc(r.MicroVMClusterToMicrovmMachine(ctx, log)),
		).
		Watches(
			&source.Kind{Type: &clusterv1.Cluster{}},
			handler.EnqueueRequestsFromMapFunc(clusterToObjectFunc),
			builder.WithPredicates(predicates.ClusterUnpausedAndInfrastructureReady(log)),
		)

	if err := builder.Complete(r); err != nil {
		return fmt.Errorf("creating microvm machine controller: %w", err)
	}

	return nil
}

// MicroVMClusterToMicrovmMachine is called when there is a change to a
// MicrovmCluster (which this controller is watching for changes to, see
// SetupWithManager). Its job is to identify the MicrovmMachines for the changed
// MicrovmCluster and queue requests (via controller-runtime) for those machines
// to be reconciled so that they can take into account any changes that are
// relevant at the MicrovmCluster level.
func (r *MicrovmMachineReconciler) MicroVMClusterToMicrovmMachine(
	ctx context.Context,
	log logr.Logger,
) handler.MapFunc {
	return func(o client.Object) []ctrl.Request {
		mvmCluster, ok := o.(*infrav1.MicrovmCluster)
		if !ok {
			log.Error(errExpectedMicrovmCluster, "failed to get microvmcluster")

			return nil
		}

		log = log.WithValues("MicrovmCluster", mvmCluster.Name, "Namespace", mvmCluster.Namespace)

		// Don't handle deleted MicrovmCluster
		if !mvmCluster.ObjectMeta.DeletionTimestamp.IsZero() {
			log.V(defaults.LogLevelDebug).Info("MicrovmCluster has a deletion timestamp, skipping mapping.")

			return nil
		}

		cluster, err := util.GetOwnerCluster(ctx, r.Client, mvmCluster.ObjectMeta)

		switch {
		case apierrors.IsNotFound(err) || cluster == nil:
			log.V(defaults.LogLevelDebug).Info("Cluster for MicrovmCluster not found, skipping mapping.")

			return nil
		case err != nil:
			log.Error(err, "Failed to get owning cluster, skipping mapping.")

			return nil
		}

		machines, err := collections.GetFilteredMachinesForCluster(ctx, r.Client, cluster)
		if err != nil {
			log.Error(err, "failed to get machines for cluster")

			return nil
		}

		var result []ctrl.Request

		for _, m := range machines.UnsortedList() {
			if m.Spec.InfrastructureRef.Name == "" {
				continue
			}

			name := client.ObjectKey{Namespace: m.Namespace, Name: m.Spec.InfrastructureRef.Name}

			result = append(result, ctrl.Request{NamespacedName: name})
		}

		return result
	}
}

func isSpecNotFound(err error) bool {
	return strings.Contains(err.Error(), "not found")
}
