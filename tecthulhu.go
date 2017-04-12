package main

import (
	"encoding/json"
	"fmt"
	"time"
)

// This module implements a module to handle communications
// with the tecthulhu device.  These devices appear to provide a WiFi
// like capability but the documentation appears to indicate a serial
// like communications protocol

// checkPortal can be used to extract status information from the portal
//
func checkPortal(device string) (results interface{}, err error) {
	// TODO Open a device of IP and get status
	err = json.Unmarshal([]byte(`"status":{"controllingFaction":"2"}`), &results)
	return results, err
}

// startPortal listens to a tecthulhu device and returns
// regular reports on the status of the portal with which it
// is associated
//
func startPortal(device string, statusC chan interface{}, errorC chan error) (err error) {

	go func() {
		poll := time.NewTicker(5 * time.Second)
		defer poll.Stop()

		for {
			select {
			case <-poll.C:
				// Perform a regular status check with the portal
				// and return the received results  to listeners using
				// the channel
				//
				// Use  a TCP and USB Serial handler function
				results, err := checkPortal(device)

				if err != nil {
					err = fmt.Errorf("portal status for %s could not be retrieved due to %s", device, err.Error())
					select {
					case errorC <- err:
					case <-time.After(2 * time.Second):
					}
					continue
				}

				select {
				case statusC <- results:
				case <-time.After(2 * time.Second):
					select {
					case errorC <- fmt.Errorf("portal status for %s was ignored", device):
					case <-time.After(2 * time.Second):
					}

				}
			}
		}
	}()

	return nil
}
