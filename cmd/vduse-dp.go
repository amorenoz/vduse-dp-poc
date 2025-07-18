package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/k8snetworkplumbingwg/govdpa/pkg/kvdpa"
)

func main() {
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("vduse%d", i)
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
		err := kvdpa.AddVduseDevice(config)
		if err != nil {
			fmt.Println("Error creating vduse device:", err.Error())
		}

		err = kvdpa.AddVdpaDevice("vduse", name)
		if err != nil {
			fmt.Println("Error creating vdpa device:", err.Error())
		}

		dev, err := kvdpa.GetVdpaDevice(name)
		if err != nil {
			fmt.Println("Error getting vdpa device:", err.Error())
			continue
		}
		err = dev.Bind(kvdpa.VhostVdpaDriver)
		if err != nil {
			fmt.Println("Error binding vdpa device:", err.Error())
		}
		fmt.Printf("vduse dev created: %s -> %s", dev.Name(), dev.VhostVdpa().Path())
	}
	fmt.Println("Created vduse devices. Ctr-C to stop")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
	fmt.Println("Deleting vduse devices:")
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("vduse%d", i)

		err := kvdpa.DeleteVdpaDevice(name)
		if err != nil {
			fmt.Println("Error deleting vdpa %s device:", name, err.Error())
		}

		err = kvdpa.DestroyVduseDevice(name)
		if err != nil {
			fmt.Println("Error deleting vduse %s device:", name, err.Error())
		}
	}
}
