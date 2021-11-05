// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package controllers

import "errors"

var errControlplaneEndpointRequired = errors.New("controlplane endpoint is required on cluster or mvmcluster")
