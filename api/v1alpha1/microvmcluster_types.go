// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const (
	// ClusterFinalizer allows ReconcileMicrovmCluster to clean up esources associated with MicrovmCluster before
	// removing it from the apiserver.
	ClusterFinalizer = "microvmcluster.infrastructure.cluster.x-k8s.io"
)

// MicrovmClusterSpec defines the desired state of MicrovmCluster.
type MicrovmClusterSpec struct {
	// ControlPlaneEndpoint represents the endpoint used to communicate with the control plane.
	//
	// See https://cluster-api.sigs.k8s.io/developer/architecture/controllers/cluster.html
	// for more details.
	//
	// +optional
	ControlPlaneEndpoint clusterv1.APIEndpoint `json:"controlPlaneEndpoint"`
}

// MicrovmClusterStatus defines the observed state of MicrovmCluster.
type MicrovmClusterStatus struct {
	// Ready indicates that the cluster is ready.
	// +optional
	// +kubebuilder:default=false
	Ready bool `json:"ready"`

	// Conditions defines current service state of the MicrovmCluster.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:path=microvmclusters,scope=Namespaced,categories=cluster-api,shortName=mvmc
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".metadata.labels.cluster\\.x-k8s\\.io/cluster-name",description="Cluster to which this MicrovmCluster belongs"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Cluster infrastructure is ready"
// +kubebuilder:printcolumn:name="ControlPlaneEndpoint",type="string",JSONPath=".spec.controlPlaneEndpoint[0]",description="API Endpoint",priority=1
// +k8s:defaulter-gen=true

// MicrovmCluster is the Schema for the microvmclusters API.
type MicrovmCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MicrovmClusterSpec   `json:"spec,omitempty"`
	Status MicrovmClusterStatus `json:"status,omitempty"`
}

// GetConditions returns the observations of the operational state of the MicrovmCluster resource.
func (r *MicrovmCluster) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

// SetConditions sets the underlying service state of the MicrovmCluster to the predescribed clusterv1.Conditions.
func (r *MicrovmCluster) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

//+kubebuilder:object:root=true

// MicrovmClusterList contains a list of MicrovmCluster.
// +k8s:defaulter-gen=true
type MicrovmClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MicrovmCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MicrovmCluster{}, &MicrovmClusterList{})
}
