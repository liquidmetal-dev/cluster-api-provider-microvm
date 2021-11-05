// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MicrovmMachineTemplateSpec defines the desired state of MicrovmMachineTemplate.
type MicrovmMachineTemplateSpec struct {
	Template MicrovmMachineTemplateResource `json:"template"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:path=microvmmachinetemplates,scope=Namespaced,categories=cluster-api,shortName=mvmmt
// +k8s:defaulter-gen=true

// MicrovmMachineTemplate is the Schema for the microvmmachinetemplates API.
type MicrovmMachineTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MicrovmMachineTemplateSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// MicrovmMachineTemplateList contains a list of MicrovmMachineTemplate.
type MicrovmMachineTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MicrovmMachineTemplate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MicrovmMachineTemplate{}, &MicrovmMachineTemplateList{})
}
