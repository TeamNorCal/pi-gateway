package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

// This module implements a module to handle communications
// with the tecthulhu device.  These devices appear to provide a WiFi
// like capability but the documentation appears to indicate a serial
// like communications protocol

type resonator struct {
	Position string `json:"position"`
	Level    int    `json:"level"`
	Health   int    `json:"health"`
	Owner    string `json:"owner"`
}

type status struct {
	Title              string      `json:"title"`
	Owner              string      `json:"owner"`
	Level              int         `json:"level"`
	Health             int         `json:"health"`
	ControllingFaction string      `json:"controllingFaction"`
	Resonators         []resonator `json:"resonators"`
}

type portalStatus struct {
	Status status `json:"status"`
}

// checkPortal can be used to extract status information from the portal
//
func checkPortal(device string) (status *portalStatus, err error) {

	body := []byte{}
	url, err := url.Parse(device)
	if err != nil {
		return nil, err
	}

	switch url.Scheme {
	case "http":
		url := "/module/status/json"
		resp, err := http.Get(device + url)
		if err != nil {
			return nil, err
		}

		body, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

	case "serial":
		return nil, fmt.Errorf("Unknown scheme %s for the tecthulhu device is not yet implemented", url.Scheme)

	default:
		return nil, fmt.Errorf("Unknown scheme %s for the tecthulhu device URI", url.Scheme)
	}

	err = json.Unmarshal(body, &status)
	return status, err
}

func getStatus(device string, statusC chan *portalStatus, errorC chan error) {
	// Perform a regular status check with the portal
	// and return the received results  to listeners using
	// the channel
	//
	// Use  a TCP and USB Serial handler function
	status, err := checkPortal(device)

	if err != nil {
		err = fmt.Errorf("portal status for %s could not be retrieved due to %s", device, err.Error())
		select {
		case errorC <- err:
		case <-time.After(2 * time.Second):
		}
		return
	}

	select {
	case statusC <- status:
	case <-time.After(2 * time.Second):
		select {
		case errorC <- fmt.Errorf("portal status for %s was ignored", device):
		case <-time.After(2 * time.Second):
		}
	}
}

// startPortal listens to a tecthulhu device and returns
// regular reports on the status of the portal with which it
// is associated
//
func startPortals(portals []string, statusC chan *portalStatus, errorC chan error, quitC chan bool) (err error) {

	poll := time.NewTicker(10 * time.Second)
	defer poll.Stop()

	for {
		select {
		case <-poll.C:
			for _, address := range portals {
				getStatus(address, statusC, errorC)
			}
		case <-quitC:
			return
		}
	}
}
