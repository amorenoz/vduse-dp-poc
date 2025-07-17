package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/k8snetworkplumbingwg/govdpa/pkg/kvdpa"
)

func main() {
	for i := 0; i < 20; i++ {
		virtioConfig := kvdpa.VirtioNetConf{}
		virtioConfig.MaxVirtqueuePairs = 1
		config := kvdpa.VduseDevConfig{
			Name:     fmt.Sprintf("vduse%d", i),
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
	}
	fmt.Println("Created vduse devices. Ctr-C to stop")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	<-c
	fmt.Println("Deleting vduse devices:")
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("vduse%d", i)
		err := kvdpa.DestroyVduseDevice(fmt.Sprintf("vduse%d", i))
		if err != nil {
			fmt.Println("Error deleting vduse %s device:", name, err.Error())
		}
	}
}
