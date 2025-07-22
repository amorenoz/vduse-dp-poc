package pool

import (
	"errors"
	"fmt"
	"os"

	"github.com/container-orchestrated-devices/container-device-interface/pkg/cdi"
	cdiSpecs "github.com/container-orchestrated-devices/container-device-interface/specs-go"
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
