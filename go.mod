module github.com/amorenoz/vduse-dp-poc

go 1.24.0

toolchain go1.24.5

require (
	github.com/container-orchestrated-devices/container-device-interface v0.5.4
	github.com/k8snetworkplumbingwg/govdpa v0.1.4
	github.com/sirupsen/logrus v1.8.1
)

require (
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/opencontainers/runc v1.1.2 // indirect
	github.com/opencontainers/runtime-spec v1.1.0 // indirect
	github.com/opencontainers/runtime-tools v0.9.1-0.20221107090550-2e043c6bd626 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	github.com/syndtr/gocapability v0.0.0-20200815063812-42c35b437635 // indirect
	github.com/vishvananda/netlink v1.3.1 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
	golang.org/x/mod v0.19.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

replace github.com/k8snetworkplumbingwg/govdpa => github.com/amorenoz/govdpa v0.0.0-20250721102101-b503056206fa
