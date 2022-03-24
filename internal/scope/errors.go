// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package scope

import "errors"

var (
	errClusterRequired        = errors.New("cluster required to create scope")
	errMicrovmClusterRequired = errors.New("microvm cluster required to create scope")

	errMachineRequired       = errors.New("machine required to create scope")
	errMicrovmMachineRequied = errors.New("microvm machine required to create scope")

	errClientRequired = errors.New("controller-runtime client required to create scope")

	errMissingBootstrapDataSecret = errors.New("missing bootstrap data secret")
	errMissingBootstrapSecretKey  = errors.New("missing bootstrap secrey value key")

	errFailureDomainNotFound = errors.New("no failure domains found on the cluster")
)
