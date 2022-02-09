// Copyright 2022 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha1

import (
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

type ExternalLoadBalancerEndpoint struct {
	// The hostname on which the API server is serving.
	// +required
	Host string `json:"host"`

	// The port on which the API server is serving.
	// +optional
	// +kubebuilder:default=6443
	Port int32 `json:"port"`
}

func (ep *ExternalLoadBalancerEndpoint) String() string {
	port := strconv.Itoa(int(ep.Port))

	return ep.Host + ":" + port
}

// ExternalLoadBalancerSpec defines the desired state for a ExternalLoadBalancer.
type ExternalLoadBalancerSpec struct {
	// Endpoint represents the endpoint for the load balancer. This endpoint will
	// be tested to see if its available.
	Endpoint    ExternalLoadBalancerEndpoint `json:"endpoint"`
	ClusterName string                       `json:"clusterName"`
}

type ExternalLoadBalancerStatus struct {
	// Ready indicates that the load balancer is ready.
	// +optional
	// +kubebuilder:default=false
	Ready bool `json:"ready"`
	// Conditions defines current state of the ExternalLoadBalancer.
	// +optional
	Conditions clusterv1.Conditions `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:resource:path=externalloadbalancers,scope=Namespaced,categories=cluster-api,shortName=extlb
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.ready",description="Load balancer is ready"
// +kubebuilder:printcolumn:name="ControlPlaneEndpoint",type="string",JSONPath=".spec.controlPlaneEndpoint[0]",description="API Endpoint",priority=1

// ExternalLoadBalancer is the schema for a external load balancer.
type ExternalLoadBalancer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExternalLoadBalancerSpec   `json:"spec,omitempty"`
	Status ExternalLoadBalancerStatus `json:"status,omitempty"`
}

// GetConditions returns the observations of the operational state of the ExternalLoadBalancer resource.
func (r *ExternalLoadBalancer) GetConditions() clusterv1.Conditions {
	return r.Status.Conditions
}

// SetConditions sets the underlying service state of the ExternalLoadBalancer to the predescribed clusterv1.Conditions.
func (r *ExternalLoadBalancer) SetConditions(conditions clusterv1.Conditions) {
	r.Status.Conditions = conditions
}

//+kubebuilder:object:root=true

// ExternalLoadBalancerList contains a list of ExternalLoadBalancers.
// +k8s:defaulter-gen=true
type ExternalLoadBalancerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MicrovmCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExternalLoadBalancer{}, &ExternalLoadBalancerList{})
}
