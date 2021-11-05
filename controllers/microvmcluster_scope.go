package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	infrav1 "github.com/weaveworks/cluster-api-provider-microvm/api/v1alpha1"
)

type clusterReconcileScope struct {
	cluster     *clusterv1.Cluster
	mvmCluster  *infrav1.MicrovmCluster
	patchHelper *patch.Helper
	log         logr.Logger
	client      client.Client
}

func (crs *clusterReconcileScope) reconcile(ctx context.Context) (requeue bool, err error) {
	crs.log.Info("Reconciling MicrovmCluster")

	controllerutil.AddFinalizer(crs.mvmCluster, infrav1.ClusterFinalizer)
	if err := crs.patchHelper.Patch(ctx, crs.mvmCluster); err != nil {
		return false, fmt.Errorf("patching mvmcluster after adding finalizer: %w", err)
	}

	//TODO: set failure domans #14

	crs.mvmCluster.Status.Ready = true

	//TODO: work out the logic for the control plane endpoint. Wait for a control plane
	// machine to be available.
	/*crs.mvmCluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
		Host: "",
		Port: 6443,
	}
	conditions.MarkTrue(crs.mvmCluster, infrav1.LoadBalancerAvailableCondition)*/

	if err := crs.patchHelper.Patch(ctx, crs.mvmCluster); err != nil {
		return false, fmt.Errorf("patching mvmcluster: %w", err)
	}

	return false, nil
}

func (crs *clusterReconcileScope) reconcileDelete(ctx context.Context) error {
	crs.log.Info("Reconciling MicrovmCluster delete")

	//TODO: add any delete

	controllerutil.RemoveFinalizer(crs.mvmCluster, infrav1.ClusterFinalizer)

	if err := crs.patchHelper.Patch(ctx, crs.mvmCluster); err != nil {
		return fmt.Errorf("patching mvmcluster: %w", err)
	}

	return nil
}
