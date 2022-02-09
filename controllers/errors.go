// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package controllers

import "errors"

var (
	errExternalLoadBalancerEndpointRefRequired = errors.New("endpointRef is required on mvmcluster")
	errClientFactoryFuncRequired               = errors.New("factory function required to create grpc client")
	errMicrovmFailed                           = errors.New("microvm is in a failed state")
	errMicrovmUnknownState                     = errors.New("microvm is in an unknown/unsupported state")
	errExpectedMicrovmCluster                  = errors.New("expected microvm cluster")
	errNoPlacement                             = errors.New("no placement specified")
	errInvalidLoadBalancerResponseStatusCode   = errors.New("endpoint returned a 5XX status code")
)
