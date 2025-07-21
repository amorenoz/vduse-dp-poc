package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/k8snetworkplumbingwg/govdpa/pkg/kvdpa"

	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	cdiSpecs "github.com/container-orchestrated-devices/container-device-interface/specs-go"
)

const (
	resourcePrefix = "vduse.io"
	poolName       = "default"
	resourceKind   = "vduse"
	cdiVersion     = "0.6.0"
	specFileName   = "vduse-devices.json"
	specDir        = "/var/run/cdi"
	numDevices     = 20
)

func createVduseDevice(name string) (*cdiSpecs.Device, error) {
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
	fmt.Printf("%s: Adding vduse device\n", name)
	err := kvdpa.AddVduseDevice(config)
	if err != nil {
		return nil, fmt.Errorf("Error creating vduse device: %v", err)
	}

	fmt.Printf("%s: Adding vdpa device bia exec\n", name)
	err = kvdpa.AddVdpaDevice("vduse", name)
	if err != nil {
		return nil, fmt.Errorf("error creating vdpa device: %v", err)
	}

	fmt.Printf("%s: Getting vdpa device\n", name)
	dev, err := kvdpa.GetVdpaDevice(name)
	if err != nil {
		return nil, fmt.Errorf("error getting vdpa device: %v", err)
	}

	fmt.Printf("%s: Binding vdpa device\n", name)
	err = dev.Bind(kvdpa.VhostVdpaDriver)
	if err != nil {
		return nil, fmt.Errorf("error binding vdpa device: %v", err)
	}
	vhostVdpaPath := dev.VhostVdpa().Path()

	// Generate CDI spec for device
	edits := cdiSpecs.ContainerEdits{
		// Add the device node to the container.
		// The container path will be the same as the host path.
		DeviceNodes: []*cdiSpecs.DeviceNode{
			{
				Path:        vhostVdpaPath,
				HostPath:    vhostVdpaPath,
				Type:        "c",
				Permissions: "rw",
			},
		},
	}

	devSpec := cdiSpecs.Device{
		Name:           name,
		ContainerEdits: edits,
	}
	return &devSpec, nil
}

func main() {
	spec := cdiSpecs.Spec{
		Version: cdiSpecs.CurrentVersion,
		Kind:    fmt.Sprintf("%s/%s", resourcePrefix, resourceKind),
		Devices: []cdiSpecs.Device{},
	}

	for i := 0; i < numDevices; i++ {
		name := fmt.Sprintf("vduse%d", i)
		devSpec, err := createVduseDevice(name)
		if err != nil {
			fmt.Printf("%s: Error creating device: %s\n", name, err.Error())
			continue
		}

		annoKey, err := cdi.AnnotationKey(resourcePrefix, resourceKind)
		if err != nil {
			fmt.Printf("error annotation key %v\n", err)

		}
		annoVal, err := cdi.AnnotationValue([]string{cdi.QualifiedName(resourcePrefix, resourceKind, name)})
		if err != nil {
			fmt.Printf("error annotation val %v\n", err)

		}

		fmt.Printf("vduse dev created: %s. Annotation: %s=\"%s\"\n", devSpec.Name,
			annoKey, annoVal)
		spec.Devices = append(spec.Devices, *devSpec)
	}

	cdiName, err := cdi.GenerateNameForSpec(&spec)
	if err != nil {
		fmt.Printf("Error generating name for spec  %v\n", err)
		os.Exit(1)
	}

	cdiName = fmt.Sprintf("%s-%s", cdiName, poolName)
	fmt.Printf("\nWriting CDI spec to %s\n", cdiName)

	if err := os.MkdirAll(specDir, 0755); err != nil {
		fmt.Printf("Error creating CDI spec directory %s: %v\n", specDir, err)
		os.Exit(1)
	}

	if err := cdi.GetRegistry().SpecDB().WriteSpec(&spec, fmt.Sprintf("%s-%s.json", cdiName, poolName)); err != nil {
		fmt.Printf("Cannot create CDI json %v", err)
		deleteAllVduseDevices()
		os.Exit(-1)
	}

	fmt.Println("Done created vduse devices. Ctr-C to stop")
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-c
	fmt.Println("Deleting vduse devices:")
	deleteAllVduseDevices()
	if err := cdi.GetRegistry().SpecDB().RemoveSpec(fmt.Sprintf("%s-%s", cdiName, poolName)); err != nil {
		fmt.Printf("Cannot delete CDI json %v", err)
		deleteAllVduseDevices()
		os.Exit(-1)
	}
}

func deleteVduseDevice(name string) error {
	err := kvdpa.DeleteVdpaDevice(name)
	if err != nil {
		fmt.Println("Error deleting vdpa %s device:", name, err.Error())
	}

	err = kvdpa.DestroyVduseDevice(name)
	if err != nil {
		fmt.Println("Error deleting vduse %s device:", name, err.Error())
	}
	return err
}

func deleteAllVduseDevices() error {
	for i := 0; i < numDevices; i++ {
		name := fmt.Sprintf("vduse%d", i)
		err := deleteVduseDevice(name)
		if err != nil {
			fmt.Printf("%s: Error deleting vduse device: %s", name, err.Error())
		}
	}
	return nil
}
