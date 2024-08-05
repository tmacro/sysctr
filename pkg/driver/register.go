package driver

import (
	"context"
	"sync"
)

var drivers = make(map[string]DriverFactory)
var driverMu sync.RWMutex

func RegisterDriver(d DriverFactory) {
	driverMu.Lock()
	defer driverMu.Unlock()
	_, ok := drivers[d.Name()]
	if ok {
		panic("driver already registered: " + d.Name())
	}

	drivers[d.Name()] = d
}

func GetDriver(ctx context.Context, name string, opts map[string]any) (Driver, error) {
	driverMu.RLock()
	defer driverMu.RUnlock()
	return drivers[name].New(ctx, opts)
}
