package deviceplugin

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
	registerapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"

	"github.com/amorenoz/vduse-dp-poc/pkg/pool"
)

const (
	sockDir        = "/var/lib/kubelet/plugins_registry" // TODO: Support old path?
	updateInterval = 10
)

// Server implements the Device Plugin API
type Server struct {
	pool         *pool.Pool
	cdi          bool
	endPoint     string
	sockPath     string
	grpcServer   *grpc.Server
	termSignal   chan bool
	updateSignal chan bool
	stopMonitor  chan bool
}

func NewDevicePluginServer(pool *pool.Pool, cdi bool) *Server {
	sockName := fmt.Sprintf("%s_%s", pool.GetResourcePrefix(), pool.GetResourceName())
	sockPath := filepath.Join(sockDir, sockName)

	return &Server{
		pool:         pool,
		cdi:          cdi,
		endPoint:     sockName,
		sockPath:     sockPath,
		grpcServer:   grpc.NewServer(),
		termSignal:   make(chan bool, 1),
		stopMonitor:  make(chan bool, 1),
		updateSignal: make(chan bool),
	}
}

// Start the Device Plugin Server
func (s *Server) Start() error {
	clog := log.WithFields(log.Fields{
		"function":     "Start",
		"endpoint":     s.endPoint,
		"resourceName": s.pool.GetResourceName(),
	})
	clog.Infof("starting Device Plugin Server: %s", s.pool.GetResourceName())

	lis, err := net.Listen("unix", s.sockPath)
	if err != nil {
		clog.Errorf("error starting device plugin: %v", err)
		return err
	}

	registerapi.RegisterRegistrationServer(s.grpcServer, s)
	pluginapi.RegisterDevicePluginServer(s.grpcServer, s)

	go func() {
		err := s.grpcServer.Serve(lis)
		if err != nil {
			clog.Errorf("serving incoming requests failed: %v", err)
		}
	}()
	s.startMonitor()

	return nil
}

func (s *Server) startMonitor() {
	clog := log.WithFields(log.Fields{
		"function":     "monitor",
		"endpoint":     s.endPoint,
		"resourceName": s.pool.GetResourceName(),
	})
	for {
		select {
		case stop := <-s.stopMonitor:
			if stop {
				clog.Infof("stopping")
				return
			}
		default:
			updated, err := s.pool.Update()
			if err != nil {
				// Socket file not found; restart server
				glog.Warningf("pool update failed: %v", err)
			} else if updated {
				s.updateSignal <- true
			}
		}
		time.Sleep(time.Second * time.Duration(updateInterval))
	}
}

func (s *Server) Stop() error {
	clog := log.WithFields(log.Fields{
		"endpoint":     s.endPoint,
		"resourceName": s.pool.GetResourceName(),
	})
	clog.Infof("stopping")
	if s.grpcServer == nil {
		return nil
	}
	s.termSignal <- true
	s.grpcServer.Stop()
	s.grpcServer = nil
	return nil
}

// Generic Plugin registration API
func (s *Server) GetInfo(ctx context.Context, rqt *registerapi.InfoRequest) (*registerapi.PluginInfo, error) {
	log.Debugf("RegistrationAPI::GetInfo")
	pluginInfoResponse := &registerapi.PluginInfo{
		Type:              registerapi.DevicePlugin,
		Name:              fmt.Sprintf("%s/%s", s.pool.GetResourcePrefix(), s.pool.GetResourceName()),
		Endpoint:          filepath.Join(sockDir, s.endPoint),
		SupportedVersions: []string{"v1alpha1", "v1beta1"},
	}
	return pluginInfoResponse, nil
}

func (s *Server) NotifyRegistrationStatus(ctx context.Context,
	regstat *registerapi.RegistrationStatus) (*registerapi.RegistrationStatusResponse, error) {
	log.Debugf("RegistrationAPI::NotifyRegistrationStatus")
	if regstat.PluginRegistered {
		log.Infof("plugin: %s gets registered successfully at Kubelet", s.endPoint)
	} else {
		log.Infof("plugin: %s failed to be registered at Kubelet: %v", s.endPoint, regstat.Error)
		s.grpcServer.Stop()
	}
	return &registerapi.RegistrationStatusResponse{}, nil
}

// Device Plugin API
func (s *Server) GetDevicePluginOptions(ctx context.Context, empty *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		PreStartRequired:                false,
		GetPreferredAllocationAvailable: false,
	}, nil
}
func (s *Server) GetPreferredAllocation(ctx context.Context,
	request *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	return &pluginapi.PreferredAllocationResponse{}, nil
}

func (s *Server) PreStartContainer(ctx context.Context,
	psRqt *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (s *Server) Allocate(ctx context.Context, rqt *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resp := new(pluginapi.AllocateResponse)
	var err error = nil

	clog := log.WithFields(log.Fields{
		"apiMethod": "Allocate",
		"endpoint":  s.endPoint,
	})
	clog.Infof("called with %+v", rqt)

	for _, container := range rqt.ContainerRequests {
		containerResp := new(pluginapi.ContainerAllocateResponse)

		if s.cdi {

			containerResp.Annotations, err = s.pool.GetAnnotation(container.DevicesIDs)
			if err != nil {
				return nil, fmt.Errorf("Allocate: can't create container annotation: %w", err)
			}
		} else {
			containerResp.Devices, err = s.pool.GetDeviceSpecs(container.DevicesIDs)
			if err != nil {
				return nil, fmt.Errorf("Allocate: can't create deviceSpecs: %w", err)
			}
			containerResp.Mounts, err = s.pool.GetMounts(container.DevicesIDs)
			if err != nil {
				return nil, fmt.Errorf("Allocate: can't create mounts: %w", err)
			}
		}

		err = s.pool.StoreDeviceInfoFile(container.DevicesIDs)
		if err != nil {
			clog.Errorf("failed to store device info file for device IDs %v: %v", container.DevicesIDs, err)
			return nil, err
		}

		containerResp.Envs = s.pool.GetEnvs(container.DevicesIDs)
		resp.ContainerResponses = append(resp.ContainerResponses, containerResp)
	}
	clog.Infof("response with %+v", resp)
	return resp, nil
}

func (s *Server) ListAndWatch(empty *pluginapi.Empty, stream pluginapi.DevicePlugin_ListAndWatchServer) error {
	resp := new(pluginapi.ListAndWatchResponse)

	clog := log.WithFields(log.Fields{
		"apiMethod": "ListAndWatch",
		"endpoint":  s.endPoint,
	})
	clog.Infof("called")

	resp.Devices = s.pool.GetAPIDevices()
	clog.Infof("send devices %v", resp)
	if err := stream.Send(resp); err != nil {
		clog.Errorf("cannot update device states: %v", err)
		s.grpcServer.Stop()
		return err
	}

	for {
		select {
		case <-s.termSignal:
			// Terminate signal received; return from method call
			clog.Infof("terminate signal received")
			return nil
		case <-s.updateSignal:
			// Devices changed; send new device list
			clog.Infof("update signal receivedd")
			resp.Devices = s.pool.GetAPIDevices()
			clog.Infof("send updated devices %v", resp)
			if err := stream.Send(resp); err != nil {
				clog.Errorf("error: cannot update device states: %v", err)
				return err
			}
		}
	}
}
