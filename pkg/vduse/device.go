package vduse

import (
	cdiSpecs "github.com/container-orchestrated-devices/container-device-interface/specs-go"
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// Common information of a VDUSE device
type VduseDevice struct {
	name          string
	vhostVdpaPath string
}

// Return the device's name.
func (d *VduseDevice) Name() string {
	return d.name
}

// Generate CDI spec information for the device
func (d *VduseDevice) CdiSpecs() *cdiSpecs.Device {
	edits := cdiSpecs.ContainerEdits{
		DeviceNodes: []*cdiSpecs.DeviceNode{
			{
				Path:        d.vhostVdpaPath,
				HostPath:    d.vhostVdpaPath,
				Type:        "c",
				Permissions: "rw",
			},
		},
	}

	devSpec := cdiSpecs.Device{
		Name:           d.name,
		ContainerEdits: edits,
	}
	return &devSpec
}

// Generate VdpaDevice information for DeviceInfo file.
func (d *VduseDevice) DeviceInfo() *nettypes.VdpaDevice {
	return &nettypes.VdpaDevice{
		ParentDevice: d.name,
		Driver:       "vhost_vdpa",
		Path:         d.vhostVdpaPath,
	}
}

// Generate DevicePlugin specifications.
func (d *VduseDevice) DeviceSpecs() *pluginapi.DeviceSpec {
	return &pluginapi.DeviceSpec{
		HostPath:      d.vhostVdpaPath,
		ContainerPath: d.vhostVdpaPath,
		Permissions:   "rw",
	}
}

// Generate DevicePlugin API Device description
func (d *VduseDevice) APIDevice() *pluginapi.Device {
	return &pluginapi.Device{
		ID:     d.name,
		Health: "Healthy",
	}
}
