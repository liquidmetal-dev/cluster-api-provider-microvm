// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha1

import "k8s.io/apimachinery/pkg/util/validation/field"

func (p *Placement) Validate() []*field.Error {
	var errs field.ErrorList

	// NOTE: this will be expanded to test for the other placement options
	if p.StaticPool == nil {
		errs = append(errs, field.Forbidden(field.NewPath("spec", "placement"), "you must supply configuration for a placement option"))
	}

	return errs
}
