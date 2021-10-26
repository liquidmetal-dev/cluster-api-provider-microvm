// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MicrovmMachineSpec defines the desired state of MicrovmMachine
type MicrovmMachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of MicrovmMachine. Edit microvmmachine_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// MicrovmMachineStatus defines the observed state of MicrovmMachine
type MicrovmMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MicrovmMachine is the Schema for the microvmmachines API
type MicrovmMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MicrovmMachineSpec   `json:"spec,omitempty"`
	Status MicrovmMachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MicrovmMachineList contains a list of MicrovmMachine
type MicrovmMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MicrovmMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MicrovmMachine{}, &MicrovmMachineList{})
}
