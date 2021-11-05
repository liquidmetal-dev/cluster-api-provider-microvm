// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package scope

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/klog/klogr"

	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "github.com/weaveworks/cluster-api-provider-microvm/api/v1alpha1"
	"github.com/weaveworks/cluster-api-provider-microvm/internal/defaults"
)

var _ Scoper = &ClusterScope{}

func NewClusterScope(cluster *clusterv1.Cluster,
	microvmCluster *infrav1.MicrovmCluster,
	client client.Client, opts ...ClusterScopeOption) (*ClusterScope, error) {
	if cluster == nil {
		return nil, errClusterRequired
	}
	if microvmCluster == nil {
		return nil, errMicrovmClusterRequired
	}
	if client == nil {
		return nil, errClientRequired
	}

	patchHelper, err := patch.NewHelper(microvmCluster, client)
	if err != nil {
		return nil, fmt.Errorf("creating patch helper for microvm cluster: %w", err)
	}

	scope := &ClusterScope{
		Cluster:        cluster,
		MvmCluster:     microvmCluster,
		client:         client,
		controllerName: defaults.ManagerName,
		Logger:         klogr.New(),
		patchHelper:    patchHelper,
	}

	for _, opt := range opts {
		opt(scope)
	}

	return scope, nil
}

type ClusterScopeOption func(*ClusterScope)

func WithClusterLogger(logger logr.Logger) ClusterScopeOption {
	return func(s *ClusterScope) {
		s.Logger = logger
	}
}

func WithClusterControllerName(name string) ClusterScopeOption {
	return func(s *ClusterScope) {
		s.controllerName = name
	}
}

// ClusterScope is the scope for reconciling a cluster.
type ClusterScope struct {
	logr.Logger

	Cluster    *clusterv1.Cluster
	MvmCluster *infrav1.MicrovmCluster

	client         client.Client
	patchHelper    *patch.Helper
	controllerName string
}

// Name returns the name of the resource.
func (cs *ClusterScope) Name() string {
	return cs.MvmCluster.Name
}

// Namespace returns the resources namespace.
func (cs *ClusterScope) Namespace() string {
	return cs.MvmCluster.Namespace
}

// ClusterName returns the name of the cluster.
func (cs *ClusterScope) ClusterName() string {
	return cs.Cluster.ClusterName
}

// ControllerName returns the name of the controller that created the scope.
func (cs *ClusterScope) ControllerName() string {
	return cs.controllerName
}

// Patch persists the resource and status.
func (cs *ClusterScope) Patch() error {
	applicableConditions := []clusterv1.ConditionType{
		infrav1.LoadBalancerAvailableCondition,
	}

	conditions.SetSummary(cs.MvmCluster,
		conditions.WithConditions(applicableConditions...),
		conditions.WithStepCounterIf(cs.MvmCluster.DeletionTimestamp.IsZero()),
		conditions.WithStepCounter(),
	)

	return cs.patchHelper.Patch(
		context.TODO(),
		cs.MvmCluster,
		patch.WithOwnedConditions{Conditions: []clusterv1.ConditionType{
			clusterv1.ReadyCondition,
			infrav1.LoadBalancerAvailableCondition,
		}})
}

// Close closes the current scope persisting the resource and status.
func (cs *ClusterScope) Close() error {
	return cs.Patch()
}
