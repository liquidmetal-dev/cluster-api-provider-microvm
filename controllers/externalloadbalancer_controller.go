// Copyright 2022 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0.

package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/cluster-api/util/predicates"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	infrav1 "github.com/weaveworks/cluster-api-provider-microvm/api/v1alpha1"
	"github.com/weaveworks/cluster-api-provider-microvm/internal/defaults"
)

type ExternalLoadBalancerReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	Recorder         record.EventRecorder
	WatchFilterValue string
	HTTPClient       *http.Client
}

const (
	httpErrorStatusCode = 50
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

	defer func() {
		if err := r.Patch(ctx, loadbalancer); err != nil {
			log.Error(err, "attempting to patch loadbalancer object")
		}
	}()

	if err := r.ensureClusterOwnerRef(ctx, req, loadbalancer); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "retrieving cluster from clusterName", "clusterName", loadbalancer.ClusterName)

		return ctrl.Result{}, err
	}

	if !loadbalancer.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Info("loadbalancer being deleted, nothing to do")

		return ctrl.Result{}, nil
	}

	if err := r.sendTestRequest(ctx, loadbalancer); err != nil {
		if os.IsTimeout(err) {
			log.Error(err, "request timed out attempting to contact endpoint", "endpoint", loadbalancer.Spec.Endpoint.String())
			conditions.MarkFalse(
				loadbalancer,
				infrav1.ExternalLoadBalancerEndpointAvailableCondition,
				infrav1.ExternalLoadBalancerEndpointNotAvailableReason,
				clusterv1.ConditionSeverityInfo, "request to loadbalancer endpoint timed out",
			)

			return ctrl.Result{}, fmt.Errorf("request timed out attempting to contact endpoint: %s: %w", loadbalancer.Spec.Endpoint.String(), err)
		}

		if errors.Is(err, errInvalidLoadBalancerResponseStatusCode) {
			log.Error(err, "request to endpoint", "endpoint", loadbalancer.Spec.Endpoint.String())
			conditions.MarkFalse(
				loadbalancer,
				infrav1.ExternalLoadBalancerEndpointAvailableCondition,
				infrav1.ExternalLoadBalancerEndpointNotAvailableReason,
				clusterv1.ConditionSeverityInfo, "loadbalancer endpoint responded with error",
			)

			return ctrl.Result{}, nil
		}

		log.Error(err, "attempting to contact specified endpoint", "endpoint", loadbalancer.Spec.Endpoint.String())
		conditions.MarkFalse(
			loadbalancer,
			infrav1.ExternalLoadBalancerEndpointAvailableCondition,
			infrav1.ExternalLoadBalancerEndpointFailedReason,
			clusterv1.ConditionSeverityInfo, "request to loadbalancer endpoint failed: %s",
			err.Error(),
		)

		return ctrl.Result{}, fmt.Errorf("attempting to contact specified endpoint: %s: %w", loadbalancer.Spec.Endpoint.String(), err)
	}

	loadbalancer.Status.Ready = true
	conditions.MarkTrue(loadbalancer, infrav1.ExternalLoadBalancerEndpointAvailableCondition)

	return ctrl.Result{}, nil
}

// Patch persists the resource and status.
func (r *ExternalLoadBalancerReconciler) Patch(ctx context.Context, lb *infrav1.ExternalLoadBalancer) error {
	applicableConditions := []clusterv1.ConditionType{
		infrav1.ExternalLoadBalancerEndpointAvailableCondition,
	}

	conditions.SetSummary(lb,
		conditions.WithConditions(applicableConditions...),
		conditions.WithStepCounterIf(lb.DeletionTimestamp.IsZero()),
		conditions.WithStepCounter(),
	)

	patchHelper, err := patch.NewHelper(lb, r.Client)
	if err != nil {
		return err
	}
	if patchErr := patchHelper.Patch(
		ctx,
		lb,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1.LoadBalancerAvailableCondition,
		}}); patchErr != nil {
		return err
	}

	return nil
}

// sendTestRequest makes an HTTP call to ${KUBE_VIP_HOST}:${KUBE_VIP_PORT}/livez, which, if the loadbalancer is live,
// should reach the /livez endpoint on the Kubernetes API server.
func (r *ExternalLoadBalancerReconciler) sendTestRequest(ctx context.Context, lb *infrav1.ExternalLoadBalancer) error {
	endpoint := lb.Spec.Endpoint.String() + "/livez"
	epReq, err := http.NewRequestWithContext(ctx, http.MethodGet, lb.Spec.Endpoint.String()+"/livez", nil) // use livez endpoint
	if err != nil {
		return fmt.Errorf("creating endpoint request: %w", err)
	}

	log.Log.V(defaults.LogLevelDebug).Info("attempting request to API server livez endpoint via loadbalancer", "endpoint_address", endpoint)
	resp, err := r.HTTPClient.Do(epReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= httpErrorStatusCode {
		return errInvalidLoadBalancerResponseStatusCode
	}

	return nil
}

func (r *ExternalLoadBalancerReconciler) ensureClusterOwnerRef(ctx context.Context, req ctrl.Request, lb *infrav1.ExternalLoadBalancer) error {
	clusterNamespaceName := types.NamespacedName{
		Namespace: req.NamespacedName.Namespace,
		Name:      lb.ClusterName,
	}

	cluster := &clusterv1.Cluster{}
	if err := r.Get(ctx, clusterNamespaceName, cluster); err != nil {
		return err
	}

	lb.OwnerReferences = util.EnsureOwnerRef(lb.OwnerReferences, metav1.OwnerReference{
		APIVersion: cluster.APIVersion,
		Kind:       cluster.Kind,
		Name:       cluster.Name,
		UID:        cluster.UID,
	})

	return nil
}

func (r *ExternalLoadBalancerReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager, options controller.Options) error {
	log := ctrl.LoggerFrom(ctx)

	if r.HTTPClient == nil {
		r.HTTPClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	builder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.ExternalLoadBalancer{}).
		WithEventFilter(predicates.ResourceNotPausedAndHasFilterLabel(log, r.WatchFilterValue)).
		WithEventFilter(predicates.ResourceIsNotExternallyManaged(log)).
		Watches(
			&source.Kind{Type: &clusterv1.Cluster{}},
			handler.EnqueueRequestsFromMapFunc(
				util.ClusterToInfrastructureMapFunc(infrav1.GroupVersion.WithKind("ExternalLoadBalancer")),
			),
			builder.WithPredicates(
				predicates.ClusterUnpaused(log),
			),
		)

	if err := builder.Complete(r); err != nil {
		return fmt.Errorf("creating external loadbalancer controller: %w", err)
	}

	return nil
}
