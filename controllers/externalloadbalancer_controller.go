// Copyright 2022 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0.

package controllers

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	infrav1 "github.com/weaveworks/cluster-api-provider-microvm/api/v1alpha1"
)

type ExternalLoadBalancerReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	Recorder         record.EventRecorder
	WatchFilterValue string
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=externalloadbalancers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=externalloadbalancers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets;,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *ExternalLoadBalancerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	loadbalancer := &infrav1.ExternalLoadBalancer{}
	if err := r.Get(ctx, req.NamespacedName, loadbalancer); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "error getting externalloadbalancer", "id", req.NamespacedName)

		return ctrl.Result{}, err
	}

}
