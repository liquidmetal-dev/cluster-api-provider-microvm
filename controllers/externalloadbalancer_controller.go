// Copyright 2022 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0.

package controllers

import (
	"context"
	"net/http"
	"os"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	"sigs.k8s.io/cluster-api/util/patch"
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

const (
	httpErrorStatusCode = 500
	warningLogVerbosity = 2
	defaultHTTPTimeout  = 5 * time.Second
)

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

	if ownerRef := loadbalancer.GetOwnerReferences(); len(ownerRef) == 0 {
		// What should we do here if the OwnerReference is empty, simply requeue??
		return ctrl.Result{RequeueAfter: requeuePeriod}, nil
	}

	if !loadbalancer.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("loadbalancer being deleted, nothing to do")

		return ctrl.Result{}, nil
	}

	client := &http.Client{
		Timeout: defaultHTTPTimeout,
	}

	epReq, err := http.NewRequestWithContext(ctx, http.MethodGet, loadbalancer.Spec.Endpoint.String(), nil)
	if err != nil {
		log.Error(err, "creating endpoint request", "id", req.NamespacedName)
	}

	resp, err := client.Do(epReq)
	if err != nil {
		if os.IsTimeout(err) {
			log.Error(err, "request timed out attempting to contact endpoint", "endpoint", loadbalancer.Spec.Endpoint.String())

			return ctrl.Result{}, err
		}
		log.Error(err, "attempting to contact specified endpoint", "endpoint", loadbalancer.Spec.Endpoint.String())

		return ctrl.Result{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= httpErrorStatusCode {
		// Do we requeue here? How do we track retries, or will this be handled automatically (CrashLoopBackoff)
		log.V(warningLogVerbosity).Info("endpoint returned a 5XX status code", "endpoint", loadbalancer.Spec.Endpoint.String())

		return ctrl.Result{}, nil
	}

	loadbalancer.Status.Ready = true

	defer func() {
		if err := r.Patch(loadbalancer); err != nil {
			log.Error(err, "attempting to patch loadbalancer object")
		}
	}()

	return ctrl.Result{}, nil
}

func (r *ExternalLoadBalancerReconciler) Patch(lb *infrav1.ExternalLoadBalancer) error {
	patchHelper, err := patch.NewHelper(lb, r.Client)
	if err != nil {
		return err
	}
	if patchErr := patchHelper.Patch(context.TODO(), lb); patchErr != nil {
		return err
	}

	return nil
}
