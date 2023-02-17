// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func (p *Placement) Validate() []*field.Error {
	var errs field.ErrorList

	// This will be expanded to test for the other placement options.
	if p.StaticPool == nil {
		fieldPath := field.NewPath("spec", "placement")
		errs = append(errs, field.Forbidden(fieldPath, "you must supply configuration for a placement option"))
	}

	return errs
}

func (s *MicrovmSpec) Validate() []*field.Error {
	var errs field.ErrorList

	if s.OsVersion == "" {
		if s.RootVolume.ID == "" {
			errs = append(errs,
				field.Required(field.NewPath("spec", "rootVolume", "id"), "must be set if osVersion is omitted"))
		}

		if s.RootVolume.Image == "" {
			errs = append(errs,
				field.Required(field.NewPath("spec", "rootVolume", "image"), "must be set if osVersion is omitted"))
		}
	}

	if s.KernelVersion == "" {
		if s.Kernel.Image == "" {
			errs = append(errs,
				field.Required(field.NewPath("spec", "kernel", "image"), "must be set if kernelVersion is omitted"))
		}

		if s.Kernel.Filename == "" {
			errs = append(errs,
				field.Required(field.NewPath("spec", "kernel", "filename"), "must be set if kernelVersion is omitted"))
		}
	}

	return errs
}
