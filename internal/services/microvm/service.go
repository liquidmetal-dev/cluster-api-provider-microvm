// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package microvm

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/weaveworks/cluster-api-provider-microvm/internal/defaults"
	"github.com/weaveworks/cluster-api-provider-microvm/internal/scope"
	"github.com/yitsushi/macpot"
	"k8s.io/utils/pointer"

	flintlockv1 "github.com/weaveworks/flintlock/api/services/microvm/v1alpha1"
	flintlocktypes "github.com/weaveworks/flintlock/api/types"
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

func (s *Service) Create(ctx context.Context) error {
	s.scope.V(defaults.LogLevelDebug).Info("Creating microvm", "machine-name", s.scope.Name(), "cluster-name", s.scope.ClusterName())

	apiMicroVM := convertToFlintlockAPI(s.scope)

	bootstrapData, err := s.scope.GetRawBootstrapData()
	if err != nil {
		return nil
	}
	apiMicroVM.Metadata["user-data"] = base64.StdEncoding.EncodeToString(bootstrapData)

	for i := range apiMicroVM.Interfaces {
		iface := apiMicroVM.Interfaces[i]

		if iface.GuestMac == nil || *iface.GuestMac == "" {
			mac, err := macpot.New(macpot.AsLocal())
			if err != nil {
				return fmt.Errorf("creating mac address client: %w", err)
			}

			iface.GuestMac = pointer.String(mac.ToString())
		}
	}

	input := &flintlockv1.CreateMicroVMRequest{
		Microvm: apiMicroVM,
	}

	_, err = s.client.CreateMicroVM(ctx, input)
	if err != nil {
		return err
	}

	s.scope.V(defaults.LogLevelDebug).Info("Successfully created microvm", "machine-name", s.scope.Name(), "cluster-name", s.scope.ClusterName())

	return nil
}

func (s *Service) Get(ctx context.Context) (*flintlocktypes.MicroVM, error) {
	s.scope.V(defaults.LogLevelDebug).Info("Getting microvm for machine", "machine-name", s.scope.Name(), "cluster-name", s.scope.ClusterName())

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
