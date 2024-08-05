package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

type DriverMap map[string]json.RawMessage

var drivers = make(map[string]DriverInfo)
var driverMu sync.RWMutex

func RegisterDriver(drv Driver) {
	mod := drv.DriverInfo()

	if mod.ID == "" {
		panic("module ID is empty")
	}

	if mod.New == nil {
		panic("module constructor is nil")
	}

	if val := mod.New(); val == nil {
		panic("module constructor returned nil")
	}

	driverMu.Lock()
	defer driverMu.Unlock()

	if _, ok := drivers[mod.ID]; ok {
		panic("module already registered: " + mod.ID)
	}

	drivers[mod.ID] = mod
}

func LoadDriver(ctx context.Context, id string, config json.RawMessage) (Driver, error) {
	driverMu.RLock()
	modInfo, ok := drivers[id]
	driverMu.RUnlock()

	if !ok {
		return nil, errors.New("driver not found: " + id)
	}

	drv := modInfo.New()

	if rv := reflect.ValueOf(drv); rv.Kind() != reflect.Ptr {
		panic("module constructor returned non-pointer")
	}

	if len(config) > 0 {
		if err := strictUnmarshal(config, &drv); err != nil {
			return nil, err
		}
	}

	if p, ok := drv.(Provisioner); ok {
		if err := p.Provision(ctx); err != nil {
			if d, ok := drv.(Destructor); ok {
				destErr := d.Destroy(ctx)
				if destErr != nil {
					return nil, fmt.Errorf("failed destroy after provisioning error: %v, provisioning error: %v", destErr, err)
				}
			}
			return nil, fmt.Errorf("failed provisioning: %v", err)
		}
	}

	return drv, nil
}

func strictUnmarshal(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
