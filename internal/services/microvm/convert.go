// Copyright 2021 Weaveworks or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MPL-2.0

package microvm

import (
	"encoding/base64"
	"fmt"

	flintlocktypes "github.com/weaveworks/flintlock/api/types"
	flcloudinit "github.com/weaveworks/flintlock/client/cloudinit"
	"gopkg.in/yaml.v3"

	"github.com/weaveworks/cluster-api-provider-microvm/api/v1alpha1"
	"github.com/weaveworks/cluster-api-provider-microvm/internal/scope"
)

const platformLiquidMetal = "liquid_metal"

func convertToFlintlockAPI(machineScope *scope.MachineScope) (*flintlocktypes.MicroVMSpec, error) {
	mvmSpec := machineScope.MvmMachine.Spec

	apiVM := &flintlocktypes.MicroVMSpec{
		Id:        machineScope.Name(),
		Namespace: machineScope.Namespace(),
		Labels: map[string]string{
			"cluster-name": machineScope.ClusterName(),
		},
		Vcpu:       int32(mvmSpec.VCPU),
		MemoryInMb: int32(mvmSpec.MemoryMb),
		Kernel: &flintlocktypes.Kernel{
			Image:            mvmSpec.Kernel.Image,
			Cmdline:          mvmSpec.KernelCmdLine,
			AddNetworkConfig: true,
			Filename:         &mvmSpec.Kernel.Filename,
		},
		RootVolume: &flintlocktypes.Volume{
			Id:         "root",
			IsReadOnly: mvmSpec.RootVolume.ReadOnly,
			MountPoint: mvmSpec.RootVolume.MountPoint,
			Source: &flintlocktypes.VolumeSource{
				ContainerSource: &mvmSpec.RootVolume.Image,
			},
		},
		Metadata: map[string]string{},
	}

	if mvmSpec.Initrd != nil {
		apiVM.Initrd = &flintlocktypes.Initrd{
			Image:    mvmSpec.Initrd.Image,
			Filename: &mvmSpec.Initrd.Filename,
		}
	}

	apiVM.AdditionalVolumes = []*flintlocktypes.Volume{}
	for i := range mvmSpec.AdditionalVolumes {
		volume := mvmSpec.AdditionalVolumes[i]

		apiVM.AdditionalVolumes = append(apiVM.AdditionalVolumes, &flintlocktypes.Volume{
			Id:         volume.ID,
			IsReadOnly: volume.ReadOnly,
			MountPoint: volume.MountPoint,
			Source: &flintlocktypes.VolumeSource{
				ContainerSource: &volume.Image,
			},
		})
	}

	apiVM.Interfaces = []*flintlocktypes.NetworkInterface{}
	for i := range mvmSpec.NetworkInterfaces {
		iface := mvmSpec.NetworkInterfaces[i]

		apiIface := &flintlocktypes.NetworkInterface{
			GuestDeviceName: iface.GuestDeviceName,
			GuestMac:        &iface.GuestMAC,
		}

		if iface.Address != "" {
			apiIface.Address = &iface.Address
		}

		switch iface.Type {
		case v1alpha1.IfaceTypeMacvtap:
			apiIface.Type = flintlocktypes.NetworkInterface_MACVTAP
		case v1alpha1.IfaceTypeTap:
			apiIface.Type = flintlocktypes.NetworkInterface_TAP
		}

		apiVM.Interfaces = append(apiVM.Interfaces, apiIface)
	}

	userMetadata := flcloudinit.Metadata{
		InstanceID:    fmt.Sprintf("%s/%s", machineScope.Namespace(), machineScope.Name()),
		LocalHostname: machineScope.Name(),
		Platform:      platformLiquidMetal,
		ClusterName:   machineScope.ClusterName(),
	}

	userMeta, err := yaml.Marshal(userMetadata)
	if err != nil {
		return apiVM, fmt.Errorf("unable to marshal metadata: %w", err)
	}

	apiVM.Metadata["meta-data"] = base64.StdEncoding.EncodeToString(userMeta)

	return apiVM, nil
}
