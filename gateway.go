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

			factionChange := lastState[state.Status.Title].Status.ControllingFaction != state.Status.ControllingFaction

			// Detect any portal changes
			//
			/**
						healthChange := lastState[state.Status.Title].Status.Health - state.Status.Health
						levelChange := lastState[state.Status.Title].Status.Level - state.Status.Level

						ownerChange := ""
						if lastState[state.Status.Title].Status.Owner != state.Status.Owner {
							ownerChange = state.Status.Owner
						}
			// Dont track individual resonator changes yet
			**/

			// Process the state updates into arduino CMDs and then send these to
			// the arduinos that are listening and our associated with the home portal
			// in any functional capacity
			if homePortal == state.Status.Title {
				cmd := []byte{}
				switch state.Status.ControllingFaction {
				case "0":
					if factionChange {
						cmd = append(cmd, 'N')
					} else {
						cmd = append(cmd, 'n')
					}
				case "1":
					if factionChange {
						cmd = append(cmd, 'E')
					} else {
						cmd = append(cmd, 'e')
					}
				case "2":
					if factionChange {
						cmd = append(cmd, 'R')
					} else {
						cmd = append(cmd, 'r')
					}
				}

				// Now dump out resonator levels, one character for each, and record the health values
				resCmd := []byte{'0', '0', '0', '0', '0', '0', '0', '0'}
				resHealth := []string{"0", "0", "0", "0", "0", "0", "0", "0"}
				// Translate an ascii compass point to a position in the resonators array
				resPositionMap := map[string]int{"N": 0, "NE": 1, "E": 2, "SE": 3, "S": 4, "SW": 5, "W": 6, "NW": 7}

				for _, res := range state.Status.Resonators {
					if position, ok := resPositionMap[res.Position]; ok {
						// After we have the position set the character in the resCmd for that
						// position to the single ASCII digit the represents the level of the
						// resonator
						resCmd[position] = strconv.Itoa(res.Level)[0]
						resHealth[position] = strconv.Itoa(res.Health)
					}
				}

				cmd = append(cmd, resCmd...)
				cmd = append(cmd, ':')
				cmd = append(cmd, []byte(strconv.FormatInt(int64(state.Status.Health), 10))...)
				for _, health := range resHealth {
					cmd = append(cmd, ':')
					cmd = append(cmd, []byte(health)...)
				}
				cmd = append(cmd, ':')

				// After printing the overall health output the per resonator health wih delimiters
				cmd = append(cmd, '\n')

				for _, device := range arduinos[homePortal] {
					response, err := device.sendCmd(cmd)
					if err != nil {
						logW.Warn(fmt.Sprintf("cmd %q sent to device %s role '%s' got an error %s", cmd, device.devName, device.role, err.Error()))
						continue
					}
					logW.Info(fmt.Sprintf("sent cmd %q to device %s role '%s' and got response %s", cmd, device.devName, device.role, response))
				}
			}

			// Save the new state as the last known state
			lastState[state.Status.Title] = state

		case <-quitC:
			return
		}
	}
}
