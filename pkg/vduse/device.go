package vduse

import (
	cdiSpecs "github.com/container-orchestrated-devices/container-device-interface/specs-go"
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
