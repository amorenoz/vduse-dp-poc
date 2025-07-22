package vduse

import (
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/k8snetworkplumbingwg/govdpa/pkg/kvdpa"
)

type VduseManager struct {
}

func NewVduseManager() *VduseManager {
	return &VduseManager{}
}

// Start the VduseManager
func (v *VduseManager) Start() error {
	log.Debugf("Starting VduseManager")
	return v.disableAutoProbe()
}

// Stop the VduseManager
func (v *VduseManager) Stop() error {
	return nil
}

// Creates a VDUSE device
func (v *VduseManager) CreateDevice(name string) (*VduseDevice, error) {
	return v.createVduseDevice(name)
}

func (v *VduseManager) DeleteDevice(name string) error {
	return v.deleteVduseDevice(name)
}

func (v *VduseManager) disableAutoProbe() error {
	err := os.WriteFile(filepath.Join("/", "sys", "bus", "vdpa", "drivers_autoprobe"),
		[]byte("0\n"), os.FileMode(os.O_SYNC))
	if err != nil {
		return fmt.Errorf("Failed to disable vdpa autoprobe: %v\n", err)
	}
	return nil
}
func (v *VduseManager) createVduseDevice(name string) (*VduseDevice, error) {
	virtioConfig := kvdpa.VirtioNetConf{}
	virtioConfig.MaxVirtqueuePairs = 1
	config := kvdpa.VduseDevConfig{
		Name:     name,
		VendorID: 0,
		DeviceID: 1,
		Features: 0xb38009fc3,
		VQNum:    2,
		VQAlign:  4096,
		Config:   &virtioConfig,
	}

	clog := log.WithFields(log.Fields{
		"vduse_device": name,
	})

	clog.Debugf("adding vduse device")
	if err := kvdpa.AddVduseDevice(config); err != nil {
		return nil, fmt.Errorf("%s: error creating vduse device: %w", name, err)
	}

	clog.Debugf("adding vdpa device")
	if err := kvdpa.AddVdpaDevice("vduse", name); err != nil {
		return nil, fmt.Errorf("%s: error creating vdpa device: %w", name, err)
	}

	clog.Debugf("getting vdpa device")
	dev, err := kvdpa.GetVdpaDevice(name)
	if err != nil {
		return nil, fmt.Errorf("%s: error getting vdpa device: %w", name, err)
	}

	clog.Debugf("binding vdpa device")
	if err = dev.Bind(kvdpa.VhostVdpaDriver); err != nil {
		return nil, fmt.Errorf("%s: error binding vdpa device: %w", name, err)
	}
	vhostVdpaPath := dev.VhostVdpa().Path()

	return &VduseDevice{
		name:          name,
		vhostVdpaPath: vhostVdpaPath,
	}, nil
}

func (v *VduseManager) deleteVduseDevice(name string) error {
	clog := log.WithFields(log.Fields{
		"vduse_device": name,
	})
	clog.Debugf("deleting vdpa device")
	err := kvdpa.DeleteVdpaDevice(name)
	if err != nil {
		return fmt.Errorf("Error deleting vdpa %s device: %v", name, err.Error())
	}

	clog.Debugf("deleting vduse device")
	err = kvdpa.DestroyVduseDevice(name)
	if err != nil {
		return fmt.Errorf("Error deleting vduse %s device: %v", name, err.Error())
	}
	return err
}
