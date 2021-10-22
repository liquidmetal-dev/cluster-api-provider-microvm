// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MicrovmClusterSpec defines the desired state of MicrovmCluster.
type MicrovmClusterSpec struct {
	// Foo is an example field of MicrovmCluster. Edit microvmcluster_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// MicrovmClusterStatus defines the observed state of MicrovmCluster.
type MicrovmClusterStatus struct{}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MicrovmCluster is the Schema for the microvmclusters API.
type MicrovmCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MicrovmClusterSpec   `json:"spec,omitempty"`
	Status MicrovmClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MicrovmClusterList contains a list of MicrovmCluster.
type MicrovmClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MicrovmCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MicrovmCluster{}, &MicrovmClusterList{})
}
