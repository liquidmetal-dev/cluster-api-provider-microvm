// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/errors"
)

const (
	// MachineFinalizer allows ReconcileMicrovmMachine to clean up resources associated with MicrovmMachine
	// before removing it from the apiserver.
	MachineFinalizer = "microvmmachine.infrastructure.cluster.x-k8s.io"
)

// MicrovmMachineSpec defines the desired state of MicrovmMachine.
type MicrovmMachineSpec struct {
	MicrovmSpec `json:",inline"`

	// SSHPublicKey is an SSH public key that will be used with the default user on this
	// machine. If specified it will take precedence over any SSH key specified at
	// the cluster level.
	// +optional
	SSHPublicKey string `json:"sshPublicKey,omitempty"`

	// ProviderID is the unique identifier as specified by the cloud provider.
	ProviderID *string `json:"providerID,omitempty"`
}

// MicrovmMachineStatus defines the observed state of MicrovmMachine.
type MicrovmMachineStatus struct {
	// Ready is true when the provider resource is ready.
	// +optional
	// +kubebuilder:default=false
	Ready bool `json:"ready"`

	// VMState indicates the state of the microvm.
	VMState *VMState `json:"vmState,omitempty"`

	// Addresses contains the microvm associated addresses.
	Addresses []clusterv1.MachineAddress `json:"addresses,omitempty"`

	// FailureReason will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a succinct value suitable
	// for machine interpretation.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureReason *errors.MachineStatusError `json:"failureReason,omitempty"`

	// FailureMessage will be set in the event that there is a terminal problem
	// reconciling the Machine and will contain a more verbose string suitable
	// for logging and human consumption.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the Machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of Machines
	// can be added as events to the Machine object and/or logged in the
	// controller's output.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`

	// Conditions defines current service state of the MicrovmMachine.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MicrovmMachine is the Schema for the microvmmachines API.
type MicrovmMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MicrovmMachineSpec   `json:"spec,omitempty"`
	Status MicrovmMachineStatus `json:"status,omitempty"`
}

// GetConditions returns the observations of the operational state of the MicrovmMachine resource.
func (r *MicrovmMachine) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

// SetConditions sets the underlying service state of the MicrovmMachine to the predescribed clusterv1.Conditions.
func (r *MicrovmMachine) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

//+kubebuilder:object:root=true

// MicrovmMachineList contains a list of MicrovmMachine.
type MicrovmMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MicrovmMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MicrovmMachine{}, &MicrovmMachineList{})
}
