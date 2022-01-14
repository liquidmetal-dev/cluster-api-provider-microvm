// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package microvm

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	flintlockv1 "github.com/weaveworks/flintlock/api/services/microvm/v1alpha1"
	flintlocktypes "github.com/weaveworks/flintlock/api/types"
	"github.com/weaveworks/flintlock/client/cloudinit"
	"github.com/yitsushi/macpot"
	"google.golang.org/protobuf/types/known/emptypb"
	"gopkg.in/yaml.v2"
	"k8s.io/utils/pointer"

	"github.com/weaveworks/cluster-api-provider-microvm/internal/defaults"
	"github.com/weaveworks/cluster-api-provider-microvm/internal/scope"
)

const (
	cloudInitHeader = "#cloud-config\n"
)

type ClientFactoryFunc func(address string) (Client, error)

type Client interface {
	flintlockv1.MicroVMClient
}

type Service struct {
	scope *scope.MachineScope

	client Client
}

func New(scope *scope.MachineScope, client Client) *Service {
	return &Service{
		scope:  scope,
		client: client,
	}
}

func (s *Service) Create(ctx context.Context, providerID string) (*flintlocktypes.MicroVM, error) {
	s.scope.
		V(defaults.LogLevelDebug).
		Info("Creating microvm", "machine-name", s.scope.Name(), "cluster-name", s.scope.ClusterName())

	apiMicroVM := convertToFlintlockAPI(s.scope)

	if err := s.addMetadata(apiMicroVM, providerID); err != nil {
		return nil, fmt.Errorf("adding metadata: %w", err)
	}

	for i := range apiMicroVM.Interfaces {
		iface := apiMicroVM.Interfaces[i]

		if iface.GuestMac == nil || *iface.GuestMac == "" {
			mac, err := macpot.New(macpot.AsLocal(), macpot.AsUnicast())
			if err != nil {
				return nil, fmt.Errorf("creating mac address client: %w", err)
			}

			iface.GuestMac = pointer.String(mac.ToString())
		}
	}

	input := &flintlockv1.CreateMicroVMRequest{
		Microvm: apiMicroVM,
	}

	resp, err := s.client.CreateMicroVM(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("creating microvm: %w", err)
	}

	s.scope.
		V(defaults.LogLevelDebug).
		Info("Successfully created microvm", "machine-name", s.scope.Name(), "cluster-name", s.scope.ClusterName())

	return resp.Microvm, nil
}

func (s *Service) Get(ctx context.Context) (*flintlocktypes.MicroVM, error) {
	s.scope.
		V(defaults.LogLevelDebug).
		Info("Getting microvm for machine", "machine-name", s.scope.Name(), "cluster-name", s.scope.ClusterName())

	input := &flintlockv1.GetMicroVMRequest{
		Id:        s.scope.Name(),
		Namespace: s.scope.Namespace(),
	}

	resp, err := s.client.GetMicroVM(ctx, input)
	if err != nil {
		return nil, err
	}

	return resp.Microvm, nil
}

func (s *Service) Delete(ctx context.Context) (*emptypb.Empty, error) {
	s.scope.
		V(defaults.LogLevelDebug).
		Info("Deleting microvm for machine", "machine-name", s.scope.Name(), "cluster-name", s.scope.ClusterName())

	input := &flintlockv1.DeleteMicroVMRequest{
		Id:        s.scope.Name(),
		Namespace: s.scope.Namespace(),
	}

	return s.client.DeleteMicroVM(ctx, input)
}

func (s *Service) addMetadata(apiMicroVM *flintlocktypes.MicroVMSpec, providerID string) error {
	bootstrapData, err := s.scope.GetRawBootstrapData()
	if err != nil {
		return fmt.Errorf("getting bootstrap data for machine: %w", err)
	}

	userdata := strings.ReplaceAll(string(bootstrapData), "PROVIDER_ID", providerID)
	apiMicroVM.Metadata["user-data"] = base64.StdEncoding.EncodeToString([]byte(userdata))

	vendorData, err := s.createVendorData()
	if err != nil {
		return fmt.Errorf("creating vendor data for machine: %w", err)
	}

	apiMicroVM.Metadata["vendor-data"] = vendorData

	instanceData, err := s.createInstanceData()
	if err != nil {
		return fmt.Errorf("creating instance metadata: %w", err)
	}

	apiMicroVM.Metadata["meta-data"] = instanceData

	return nil
}

func (s *Service) createVendorData() (string, error) {
	// TODO: remove the boot command temporary fix after image-builder change #89
	vendorUserdata := &cloudinit.UserData{
		HostName:     s.scope.MvmMachine.Name,
		FinalMessage: "The Liquid Metal booted system is good to go after $UPTIME seconds",
		BootCommands: []string{
			"ln -sf /run/systemd/resolve/stub-resolv.conf /etc/resolv.conf",
		},
	}

	// TODO:  allow setting multiple keys #88
	machineSSHKey := s.scope.GetSSHPublicKey()
	if machineSSHKey != "" {
		defaultUser := cloudinit.User{
			Name: "ubuntu",
		}
		rootUser := cloudinit.User{
			Name: "root",
		}

		defaultUser.SSHAuthorizedKeys = []string{
			machineSSHKey,
		}
		rootUser.SSHAuthorizedKeys = []string{
			machineSSHKey,
		}

		vendorUserdata.Users = []cloudinit.User{defaultUser, rootUser}
	}

	data, err := yaml.Marshal(vendorUserdata)
	if err != nil {
		return "", fmt.Errorf("marshalling bootstrap data: %w", err)
	}

	dataWithHeader := append([]byte(cloudInitHeader), data...)

	return base64.StdEncoding.EncodeToString(dataWithHeader), nil
}

func (s *Service) createInstanceData() (string, error) {
	userMetadata := cloudinit.Metadata{
		InstanceID:    fmt.Sprintf("%s/%s", s.scope.Namespace(), s.scope.Name()),
		LocalHostname: s.scope.Name(),
		Platform:      platformLiquidMetal,
		ClusterName:   s.scope.ClusterName(),
	}

	userMeta, err := yaml.Marshal(userMetadata)
	if err != nil {
		return "", fmt.Errorf("unable to marshal metadata: %w", err)
	}

	return base64.StdEncoding.EncodeToString(userMeta), nil
}
