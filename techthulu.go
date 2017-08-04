package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// This module implements a module to handle communications
// with the tecthulhu device.  These devices appear to provide a WiFi
// like capability but the documentation appears to indicate a serial
// like communications protocol

type tResonator struct {
	Position string `json:"position"`
	Level    int    `json:"level"`
	Health   int    `json:"health"`
	Owner    string `json:"owner"`
}

type tStatus struct {
	Title              string      `json:"title"`
	Owner              string      `json:"owner"`
	Level              int         `json:"level"`
	Health             int         `json:"health"`
	ControllingFaction string      `json:"controllingFaction"`
	Mods               []string    `json:"mods"`
	Resonators         []resonator `json:"resonators"`
}

type tPortalStatus struct {
	State tStatus `json:"status"`
}

type tecthulhu struct {
	url     string
	statusC chan *portalStatus
	errorC  chan error
}

func (tec *tPortalStatus) Status() (state *portalStatus) {
	state = &portalStatus{
		Status: status{
			Title:              tec.State.Title,
			Owner:              tec.State.Owner,
			Level:              float32(tec.State.Level),
			Health:             float32(tec.State.Health),
			ControllingFaction: tec.State.ControllingFaction,
			Mods:               []mod{},
			Resonators:         []resonator{},
		},
	}
	for _, res := range tec.State.Resonators {
		state.Status.Resonators = append(state.Status.Resonators,
			resonator{
				Position: res.Position,
				Level:    res.Level,
				Health:   res.Health,
				Owner:    res.Owner,
			})
	}
	for i, modStr := range tec.State.Mods {
		newMod := mod{Slot: float32(i)}
		modParts := strings.Split(modStr, "-")
		if len(modParts) == 2 {
			switch modParts[1] {
			case "C":
				newMod.Rarity = "Common"
			case "R":
				newMod.Rarity = "Rare"
			case "VR":
				newMod.Rarity = "Very Rare"
			}
		}
		switch modParts[0] {
		case "FA":
			newMod.Type = "Force Amplifier"
		case "HS":
			newMod.Type = "Heat Sink"
		case "LA":
			newMod.Type = "Link Amplifier"
		case "SBUL":
			newMod.Type = "SoftBank UltraLink"
		case "MH":
			newMod.Type = "Multi-hack"
		case "PS":
			newMod.Type = "Portal Shield"
		case "AXA":
			newMod.Type = "AXA Shield"
		case "T":
			newMod.Type = "Turret"
		}
		state.Status.Mods = append(state.Status.Mods, newMod)
	}
	return state
}

// checkPortal can be used to extract status information from the portal
//
func (tec *tecthulhu) checkPortal() (status *portalStatus, err error) {

	body := []byte{}
	url, err := url.Parse(tec.url)
	if err != nil {
		return nil, err
	}

	switch url.Scheme {
	case "http":
		url := "/module/status/json"
		resp, err := http.Get(tec.url + url)
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

	// Parse into the techthulu specific format and then convert to
	// the canonical format used by the concentrator which we assume
	// is a reference format for portal data and meta data
	//
	tecStatus := &tPortalStatus{}

	err = json.Unmarshal(body, &tecStatus)
	if err != nil {
		logW.Debug(string(body))
	}
	return tecStatus.Status(), err
}

func (tec *tecthulhu) getStatus() {
	// Perform a regular status check with the portal
	// and return the received results  to listeners using
	// the channel
	//
	// Use  a TCP and USB Serial handler function
	status, err := tec.checkPortal()

	if err != nil {
		err = fmt.Errorf("portal status for %s could not be retrieved due to %s", tec.url, err.Error())
		go func() {
			select {
			case tec.errorC <- err:
			case <-time.After(500 * time.Millisecond):
				logW.Warn(fmt.Sprintf("could not send, error for ignored portal status update %s", err.Error()))
			}
		}()
		return
	}

	select {
	case tec.statusC <- status:
	case <-time.After(750 * time.Millisecond):
		go func() {
			select {
			case tec.errorC <- fmt.Errorf("portal status for %s had to be skipped", tec.url):
			case <-time.After(2 * time.Second):
				logW.Warn("could not send error for ignored portal status update")
			}
		}()
	}
}

// startPortal listens to a tecthulhu device and returns
// regular reports on the status of the portal with which it
// is associated
//
func (tec *tecthulhu) startPortals(quitC chan bool) (err error) {

	poll := time.NewTicker(2 * time.Second)
	defer poll.Stop()

	for {
		select {
		case <-poll.C:
			tec.getStatus()
		case <-quitC:
			return
		}
	}
}
