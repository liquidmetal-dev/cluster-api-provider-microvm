// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package v1alpha1

import (
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

// MicrovmMachineTemplateResource describes the data needed to create a MicrovmMachine from a template.
type MicrovmMachineTemplateResource struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	ObjectMeta clusterv1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the specification of the machine.
	Spec MicrovmMachineSpec `json:"spec"`
}

// Placement represents configuration relating to the placement of the microvms. The number of placement
// options will grow and so we need to ensure in the validation webhook that only 1 placement types
// is configured.
type Placement struct {
	// StaticPool is used to specify that static pool placement should be used.
	StaticPool *StaticPoolPlacement `json:"staticPool,omitempty"`
}

// IsSet returns true if one of the placement options has been configured.
// NOTE: this will need to be expanded as the placement options grow.
func (p *Placement) IsSet() bool {
	return p.StaticPool != nil
}

// StaticPoolPlacement represents the configuration for placing microvms across
// a pool of predefined servers.
type StaticPoolPlacement struct {
	// Hosts defines the pool of hosts that should be used when creating microvms. The hosts will
	// be supplied to CAPI (as fault domains) and it will place machines across them.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems:=1
	Hosts []MicrovmHost `json:"hosts"`
	// BasicAuthSecret is the name of the secret containing basic auth info for each
	// host listed in Hosts.
	// The secret should be created in the same namespace as the Cluster.
	// The secret should contain a data entry for each host Endpoint without the port:
	//
	// apiVersion: v1
	// kind: Secret
	// metadata:
	// 	name: mybasicauthsecret
	//	namespace: same-as-cluster
	// type: Opaque
	// data:
	// 	1.2.4.5: YWRtaW4=
	// 	myhost: MWYyZDFlMmU2N2Rm
	BasicAuthSecret string `json:"basicAuthSecret,omitempty"`
}

type MicrovmHost struct {
	// Name is an optional name for the host.
	// +optional
	Name string `json:"name,omitempty"`
	// Endpoint is the API endpoint for the microvm service (i.e. flintlock)
	// including the port.
	// +kubebuilder:validation:Required
	Endpoint string `json:"endpoint"`
	// ControlPlaneAllowed marks this host as suitable for running control plane nodes in
	// addition to worker nodes.
	// +kubebuilder:default=true
	ControlPlaneAllowed bool `json:"controlplaneAllowed"`
}

// TLSConfig represents config for connecting to TLS enabled hosts.
type TLSConfig struct {
	Cert   []byte `json:"cert"`
	Key    []byte `json:"key"`
	CACert []byte `json:"caCert"`
}
