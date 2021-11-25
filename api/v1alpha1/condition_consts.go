// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha1

import clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

// Conditions and reasons for capmvm

const (
	// LoadBalancerAvailableCondition is a condition that indicates that the API server
	// load balancer is available.
	LoadBalancerAvailableCondition clusterv1.ConditionType = "LoadBalancerAvailable"

	// LoadBalancerFailedReason is used to indicate any error with the
	// availability of the load balancer.
	LoadBalancerFailedReason = "LoadBalancerAvailabilityFailed"

	// LoadBalancerNotAvailableReason is used to indicate that the load balancer isn't available.
	LoadBalancerNotAvailableReason = "LoadBalancerNotAvailable"
)
