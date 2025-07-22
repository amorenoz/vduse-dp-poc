package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	plugin "github.com/amorenoz/vduse-dp-poc/pkg/deviceplugin"
	"github.com/amorenoz/vduse-dp-poc/pkg/pool"
	"github.com/amorenoz/vduse-dp-poc/pkg/vduse"
)

const (
	resourcePrefix = "vduse.io"
	poolName       = "default"
	resourceKind   = "vduse"
	numDevices     = 20
)

var logLevel = flag.String("log-level", "info", "the log level")
var cdi = flag.Bool("cdi", false, "whether to use cdi specs")

func main() {
	flag.Parse()
	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		fmt.Printf("invalid log level: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}
	log.SetLevel(level)

	vduseMan := vduse.NewVduseManager()
	if err := vduseMan.Start(); err != nil {
		log.Fatalf("failed to initialize VduseManager: %v", err)
	}

	pool := pool.NewPool(poolName, resourcePrefix, resourceKind, numDevices, vduseMan)
	if err := pool.Start(); err != nil {
		log.Errorf("pool initialization failed, some devices might not be fully functional: %v", err)
	}

	if err := pool.WriteCdiSpec(); err != nil {
		log.Errorf("failed to write CDI spec: %v", err)
	}

	server := plugin.NewDevicePluginServer(pool, *cdi)
	if err := server.Start(); err != nil {
		if err := pool.RemoveCdiSpec(); err != nil {
			log.Errorf("failed to remove CDI spec: %v", err)
		}
		if err := pool.Stop(); err != nil {
			log.Errorf("failed to stop pool: %v", err)
		}
		if err := vduseMan.Stop(); err != nil {
			log.Errorf("failed to stop vduseManager: %v", err)
		}
		log.Fatalf("server initialization failed: %v", err)
	}

	log.Infof("pool created. Ctr-C to stop")
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	<-c
	log.Infof("cleaning up")

	if err := server.Stop(); err != nil {
		log.Errorf("failed to stop server %v", err)
	}
	if err := pool.RemoveCdiSpec(); err != nil {
		log.Errorf("failed to remove CDI spec: %v", err)
	}
	if err := pool.Stop(); err != nil {
		log.Errorf("failed to stop pool: %v", err)
	}
	if err := vduseMan.Stop(); err != nil {
		log.Errorf("failed to stop vduseManager: %v", err)
	}
}
