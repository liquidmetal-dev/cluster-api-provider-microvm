// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha1

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ = logf.Log.WithName("mvmcluster-resource")

func (r *MicrovmCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	if err := ctrl.NewWebhookManagedBy(mgr).For(r).Complete(); err != nil {
		return fmt.Errorf("unable to setup webhook: %w", err)
	}

	return nil
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1alpha1-microvmcluster,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=microvmclusters,versions=v1alpha1,name=validation.microvmcluster.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1
// +kubebuilder:webhook:verbs=create;update,path=/mutate-infrastructure-cluster-x-k8s-io-v1alpha1-microvmcluster,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=microvmclusters,versions=v1alpha1,name=default.microvmcluster.infrastructure.cluster.x-k8s.io,sideEffects=None,admissionReviewVersions=v1beta1

var (
	_ webhook.Validator = &MicrovmCluster{}
	_ webhook.Defaulter = &MicrovmCluster{}
)

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *MicrovmCluster) ValidateCreate() (admission.Warnings, error) {
	var allErrs field.ErrorList

	allErrs = append(allErrs, r.Spec.Placement.Validate()...)

	if len(allErrs) == 0 {
		return nil, nil
	}

	return nil, apierrors.NewInvalid(r.GroupVersionKind().GroupKind(), r.Name, allErrs)
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
func (r *MicrovmCluster) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *MicrovmCluster) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

// Default satisfies the defaulting webhook interface.
func (r *MicrovmCluster) Default() {
}
