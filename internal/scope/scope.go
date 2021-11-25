// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package scope

import "github.com/go-logr/logr"

// Scoper is the interface for a scope.
type Scoper interface {
	logr.Logger

	// Name returns the name of the resource.
	Name() string
	// Namespace returns the resources namespace.
	Namespace() string
	// ClusterName returns the name of the cluster.
	ClusterName() string

	// ControllerName returns the name of the controller that created the scope.
	ControllerName() string

	// Patch persists the resource and status.
	Patch() error
	// Close closes the current scope persisting the resource and status.
	Close() error
}
