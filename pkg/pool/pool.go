package pool

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	cdiSpecs "github.com/container-orchestrated-devices/container-device-interface/specs-go"
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	nadutils "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/utils"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"

	log "github.com/sirupsen/logrus"

	"github.com/amorenoz/vduse-dp-poc/pkg/vduse"
)

const (
	specDir = "/var/run/cdi"
)

type Pool struct {
	name           string
	resourcePrefix string
	resourceKind   string
	numDevices     int
	devices        map[string]vduse.VduseDevice
	vduse          *vduse.VduseManager
	cdiName        string
}

func NewPool(name, prefix, kind string, num int, vduseMan *vduse.VduseManager) *Pool {
	return &Pool{
		name:           name,
		resourcePrefix: prefix,
		resourceKind:   kind,
		numDevices:     num,
		vduse:          vduseMan,
		devices:        make(map[string]vduse.VduseDevice, 0),
		cdiName:        "",
	}
}

func (p *Pool) GetResourceName() string {
	return p.name
}

func (p *Pool) GetResourcePrefix() string {
	return p.resourcePrefix
}

func (p *Pool) GetResourceKind() string {
	return p.resourceKind
}

func (p *Pool) Start() error {
	var errs error = nil

	for i := range p.numDevices {
		name := fmt.Sprintf("vduse%d", i)
		clog := log.WithFields(log.Fields{"vduse_device": name})

		clog.Debugf("creating device")
		dev, err := p.vduse.CreateDevice(name)
		if err != nil {
			clog.Errorf("error creating vduse device: %v\n", err)
			errs = errors.Join(errs, err)
		} else {
			p.devices[name] = *dev
		}
	}
	return errs
}

func (p *Pool) Update() (bool, error) {
	// TBD
	return true, nil
}

func (p *Pool) WriteCdiSpec() error {
	spec := cdiSpecs.Spec{
		Version: cdiSpecs.CurrentVersion,
		Kind:    fmt.Sprintf("%s/%s", p.resourcePrefix, p.resourceKind),
		Devices: make([]cdiSpecs.Device, 0),
	}

	for _, device := range p.devices {
		spec.Devices = append(spec.Devices, *device.CdiSpecs())
	}

	if err := p.setCdiName(&spec); err != nil {
		return fmt.Errorf("error creating cdi spec file: %w", err)
	}
	if err := os.MkdirAll(specDir, 0755); err != nil {
		return fmt.Errorf("error creating CDI spec directory %s: %w\n", specDir, err)
	}
	if err := cdi.GetRegistry().SpecDB().WriteSpec(&spec, p.cdiName); err != nil {
		return fmt.Errorf("cannot create CDI json: %w", err)
	}

	log.Infof("Written CDI spec to %s", p.cdiName)
	return nil
}

func (p *Pool) RemoveCdiSpec() error {
	if p.cdiName == "" {
		return nil
	}
	if err := cdi.GetRegistry().SpecDB().RemoveSpec(p.cdiName); err != nil {
		log.Warnf("cannot delete CDI json %v", err)
		return fmt.Errorf("cannot delete CDI spec file: %w", err)
	}
	return nil
}

func (p *Pool) Stop() error {
	var errs error = nil

	for name, _ := range p.devices {
		err := p.vduse.DeleteDevice(name)
		if err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}

func (p *Pool) GetAnnotation(deviceIDs []string) (map[string]string, error) {
	annotations := make(map[string]string, 0)

	annoKey, err := cdi.AnnotationKey(p.resourcePrefix, p.resourceKind)
	if err != nil {
		return nil, fmt.Errorf("error annotation key: %w\n", err)

	}
	devices := make([]string, 0)
	for _, id := range deviceIDs {
		devices = append(devices, cdi.QualifiedName(p.resourcePrefix, p.resourceKind, id))
	}
	annoVal, err := cdi.AnnotationValue(devices)
	if err != nil {
		return nil, fmt.Errorf("error annotation val %w\n", err)

	}
	annotations[annoKey] = annoVal
	return annotations, nil
}

func (p *Pool) setCdiName(spec *cdiSpecs.Spec) error {
	cdiName, err := cdi.GenerateNameForSpec(spec)
	if err != nil {
		return fmt.Errorf("error generating name for spec: %w\n", err)
	}
	p.cdiName = fmt.Sprintf("%s-%s.json", cdiName, p.name)
	return nil
}

func (p *Pool) GetDeviceSpecs(deviceIDs []string) ([]*pluginapi.DeviceSpec, error) {
	devSpecs := make([]*pluginapi.DeviceSpec, 0)

	for _, id := range deviceIDs {
		dev, ok := p.devices[id]
		if !ok {
			return nil, fmt.Errorf("GetDeviceSpecs failed, not such device: %s", id)
		}
		devSpecs = append(devSpecs, dev.DeviceSpecs())
	}
	log.WithFields(log.Fields{"deviceIDs": deviceIDs}).Infof("device specs: %v", devSpecs)
	return devSpecs, nil
}

func (p *Pool) GetMounts(deviceIDs []string) ([]*pluginapi.Mount, error) {
	mounts := make([]*pluginapi.Mount, 0)
	log.WithFields(log.Fields{"deviceIDs": deviceIDs}).Infof("mounts: %v", mounts)
	return mounts, nil
}

func (p *Pool) GetEnvs(deviceIDs []string) map[string]string {
	envs := make(map[string]string)

	key := fmt.Sprintf("%s_%s_%s", "VDUSEDEVICE", p.resourcePrefix, p.name)
	key = strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	envs[key] = strings.Join(deviceIDs, ",")

	log.WithFields(log.Fields{"deviceIDs": deviceIDs}).Infof("envs: %v", envs)
	return envs
}

func (p *Pool) GetAPIDevices() []*pluginapi.Device {
	devs := make([]*pluginapi.Device, 0)
	for _, dev := range p.devices {
		devs = append(devs, dev.APIDevice())
	}
	return devs
}

func (p *Pool) StoreDeviceInfoFile(deviceIDs []string) error {
	var devInfo nettypes.DeviceInfo

	for _, id := range deviceIDs {
		dev, ok := p.devices[id]
		if !ok {
			return fmt.Errorf("cannot store DeviceInfo file, not such device: %s", id)
		}
		devInfo = nettypes.DeviceInfo{
			Type:    nettypes.DeviceInfoTypeVDPA,
			Version: nettypes.DeviceInfoVersion,
			Vdpa:    dev.DeviceInfo(),
		}

		resource := fmt.Sprintf("%s/%s", p.resourcePrefix, p.name)
		if err := nadutils.CleanDeviceInfoForDP(resource, id); err != nil {
			return fmt.Errorf("error cleaning device-info file before writing: %w", err)
		}
		if err := nadutils.SaveDeviceInfoForDP(resource, id, &devInfo); err != nil {
			return fmt.Errorf("error creating device-info file: %w", err)
		}
	}
	return nil
}

func (p *Pool) CleanDeviceInfoFile() error {
	var errs error

	for id := range p.devices {
		resource := fmt.Sprintf("%s/%s", p.resourcePrefix, p.name)
		if err := nadutils.CleanDeviceInfoForDP(resource, id); err != nil {
			errs = errors.Join(errs, err)
		}
	}
	return errs
}
