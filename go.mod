module github.com/amorenoz/vduse-dp

go 1.24.5

require github.com/k8snetworkplumbingwg/govdpa v0.1.4

require (
	github.com/vishvananda/netlink v1.1.0 // indirect
	github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df // indirect
	golang.org/x/sys v0.1.0 // indirect
)

replace github.com/k8snetworkplumbingwg/govdpa => github.com/amorenoz/govdpa v0.0.0-20250717200646-8c267a068274
