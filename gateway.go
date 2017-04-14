package main

// This module implements a gateway that bridges between IP based JSON events and the
// ASCII line delimited control messages intended for arduinos under the control of
// the gateway
//
import (
	"fmt"
	"strconv"
)

var (
	// Record the last known state of a portal in order that transitions can be discovered
	// and a diff can be sent to the arduino so that it does not have to track changes
	// in alignment etc
	//
	lastState = map[string]*portalStatus{}
)

type resChange struct {
	factionChange string
	levelChange   int
	healthChange  int
	ownerChange   string
}

func startGateway(homePortal string, arduinos map[string]map[string]*arduino, tectC chan *portalStatus, quitC chan bool) {
	for {
		select {
		case state := <-tectC:
			// If there is not history add the fresh state as the previous state
			//
			if _, ok := lastState[state.Status.Title]; !ok {
				lastState[state.Status.Title] = state
			}

			// Detect any portal changes
			//
			/**
						factionChange := lastState[state.Status.Title].Status.ControllingFaction != state.Status.ControllingFaction
						healthChange := lastState[state.Status.Title].Status.Health - state.Status.Health
						levelChange := lastState[state.Status.Title].Status.Level - state.Status.Level

						ownerChange := ""
						if lastState[state.Status.Title].Status.Owner != state.Status.Owner {
							ownerChange = state.Status.Owner
						}
			**/
			// Dont track individual resonator changes yet

			// Process the state updates into arduino CMDs and then send these to
			// the arduinos that are listening and our associated with the home portal
			// in any functional capacity
			if homePortal == state.Status.Title {
				for node, device := range arduinos[homePortal] {
					cmd := []byte{}
					switch state.Status.ControllingFaction {
					case "1":
						cmd = append(cmd, 'e')
					case "2":
						cmd = append(cmd, 'r')
					}
					cmd = append(cmd, []byte(strconv.FormatInt(int64(state.Status.Health), 10))...)
					cmd = append(cmd, '\n')

					response, err := device.sendCmd(cmd)
					if err != nil {
						logW.Warn(fmt.Sprintf("cmd %q sent to %s device %s got an error %s", cmd, node, device.devName, err.Error()))
						continue
					}

					logW.Info(fmt.Sprintf("sent cmd %q to %s device %s and got response %s", cmd, node, device.devName, response))
				}
			}

			// Save the new state as the last known state
			lastState[state.Status.Title] = state

		case <-quitC:
			return
		}
	}
}
