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
// with 1st generation ingress gateway device.  These devices
// are provisioned using standard HTTP base JSON server and are able
// to support data feeds from multiple portals.  This file
// implements a client that supports a single HTTP resource for
// a single portal.

type resonator struct {
	Position string  `json:"position"`
	Level    float32 `json:"level"`
	Health   float32 `json:"health"`
	Owner    string  `json:"owner"`
}

type mod struct {
	Owner  string  `json:"owner"`
	Slot   float32 `json:"slot"`
	Type   string  `json:"type"`
	Rarity string  `json:"rarity"`
}

type status struct {
	Title              string      `json:"Title"`
	Description        string      `json:"description"`
	CoverImageURL      string      `json:"coverImageUrl"`
	Owner              string      `json:"owner"`
	Level              float32     `json:"level"`
	Health             float32     `json:"health"`
	ControllingFaction string      `json:"controllingFaction"`
	Mods               []mod       `json:"mods"`
	Resonators         []resonator `json:"resonators"`
}

type portalStatus struct {
	Status status `json:"externalApiPortal"`
}

type concentrator struct {
	url     string
	statusC chan *portalStatus
	errorC  chan error
}

// checkPortal can be used to extract status information from the portal
//
func (conc *concentrator) checkPortal() (status *portalStatus, err error) {

	body := []byte{}
	url, err := url.Parse(conc.url)
	if err != nil {
		return nil, err
	}

	switch url.Scheme {
	case "http":
		resp, err := http.Get(url.String())
		if err != nil {
			return nil, err
		}

		body, err = ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

	case "serial":
		return nil, fmt.Errorf("Unknown scheme %s for the concentrator device is not yet implemented", url.Scheme)

	default:
		return nil, fmt.Errorf("Unknown scheme %s for the concentrator device URI", url.Scheme)
	}

	err = json.Unmarshal(body, &status)
	if err != nil {
		logW.Debug(fmt.Sprintf("bad data %s", err))
		return nil, err
	}

	return status, err
}

func (conc *concentrator) getStatus() {
	// Perform a regular status check with the portal
	// and return the received results to listeners using
	// the channel
	//
	// Use a TCP and USB Serial handler function
	//
	status, err := conc.checkPortal()

	if err != nil {
		err = fmt.Errorf("portal status for %s could not be retrieved due to %s", conc.url, err.Error())
		go func() {
			select {
			case conc.errorC <- err:
			case <-time.After(500 * time.Millisecond):
				logW.Warn(fmt.Sprintf("could not send, error for ignored portal status update %s", err.Error()))
			}
		}()
		return
	}

	select {
	case conc.statusC <- status:
	case <-time.After(750 * time.Millisecond):
		go func() {
			select {
			case conc.errorC <- fmt.Errorf("portal status for %s had to be skipped", conc.url):
			case <-time.After(2 * time.Second):
				logW.Warn("could not send error for ignored portal status update")
			}
		}()
	}
}

// startPortal listens to a concentrator device and returns
// regular reports on the status of the portal with which it
// is associated
//
func (conc *concentrator) startPortals(quitC chan bool) (err error) {

	poll := time.NewTicker(2 * time.Second)
	defer poll.Stop()

	for {
		select {
		case <-poll.C:
			conc.getStatus()
		case <-quitC:
			return
		}
	}
}
