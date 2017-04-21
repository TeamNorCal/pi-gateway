package main

// This module implements a component that listens for new arduino style
// connections to the Pi and when they are found and can be validated then it will
// notify the gateway that a new device is ready for communications

import (
	"fmt"
	"sync"
	"time"
)

type deviceCatalog struct {
	devices map[string]map[string]*arduino
	sync.Mutex
}

var devices = deviceCatalog{
	devices: map[string]map[string]*arduino{},
}

func plugAndPlay(quitC chan bool) {

	devices.devices = map[string]map[string]*arduino{*homeTecthulhu: map[string]*arduino{}}

	candidates := map[string]map[string]bool{*homeTecthulhu: map[string]bool{}}

	for {
		for _, device := range findDevices() {
			// At the moment we only control a single portals audrinos however
			// this can be changed very simply by using multiple names here
			candidates[*homeTecthulhu][device] = true
		}

		// Get the current catalog of open working devices
		working := getRunningDevices(*homeTecthulhu)

		// Check if the device is new, and if so try starting it
		for _, device := range working {
			if _, ok := candidates[*homeTecthulhu][device.devName]; ok {
				delete(candidates[*homeTecthulhu], device.devName)
			}
		}

		for portalName, devNames := range candidates {
			for name := range devNames {

				device, err := startDevice(portalName, name)
				if err != nil {
					stopRunningDevice(portalName, name)
					continue
				}

				if func() bool {
					devices.Lock()
					defer devices.Unlock()
					if _, ok := devices.devices[portalName][name]; !ok {
						devices.devices[portalName][name] = device
						return true
					} else {
						device.close()
					}
					return false
				}() {
					logW.Info(fmt.Sprintf("arduino at %s has the role of '%s'", device.devName, device.role))
				}
			}
		}
		select {
		case <-time.After(10 * time.Second):
		case <-quitC:
			return
		}
	}
}

func getRunningDevices(portal string) (devs map[string]*arduino) {
	devices.Lock()
	defer devices.Unlock()

	found, ok := devices.devices[portal]
	if !ok {
		return map[string]*arduino{}
	}

	devs = make(map[string]*arduino, len(found))
	for deviceName, device := range found {
		devs[deviceName] = device
	}
	return devs
}

func stopRunningDevice(portal string, device string) {
	defer func() {
		recover()
	}()

	devices.Lock()
	defer devices.Unlock()

	if dev, ok := devices.devices[portal][device]; ok {
		delete(devices.devices[portal], device)
		dev.close()
	}
}
