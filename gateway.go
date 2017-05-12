package main

// This module implements a gateway that bridges between IP based JSON events and the
// ASCII line delimited control messages intended for arduinos under the control of
// the gateway
//
import (
	"fmt"
	"strconv"
	"time"
)

var (
	// Record the last known state of a portal in order that transitions can be discovered
	// and a diff can be sent to the arduino so that it does not have to track changes
	// in alignment etc
	//
	lastState = map[string]*portalStatus{}
)

func encodePercent(v int) byte {
	if v == 0 {
		return ' '
	}
	return byte(int(' ') + (v / 2))
}

func startGateway(homePortal string, tectC chan *portalStatus, ambientC chan<- string, sfxC chan<- []string, quitC chan bool) {

	// Used to trigger a manual update for the ambient noise effects
	forceAmbient := false

	// Used to track the addition and reduction in the number of resonators
	resCount := 0

	for {
		select {
		case state := <-tectC:
			// If there is not history add the fresh state as the previous state
			//
			if _, ok := lastState[state.Status.Title]; !ok {
				lastState[state.Status.Title] = state
				forceAmbient = true
			}
			// Sounds effects that are gathered as a result of state
			// and played back later
			sfxs := []string{}

			factionChange := lastState[state.Status.Title].Status.ControllingFaction != state.Status.ControllingFaction

			if factionChange {

				if resCount != 0 {
					// Trigger the res destroyed audio for the last faction to
					// own the portal
					resCount = 0
				}
				// e-loss, r-loss, n-loss
				switch lastState[state.Status.Title].Status.ControllingFaction {
				case "0":
					sfxs = append(sfxs, "n-loss")
				case "1":
					sfxs = append(sfxs, "e-loss")
				case "2":
					sfxs = append(sfxs, "r-loss")
				default:
					logW.Warn(fmt.Sprintf("unknown faction '%s'", state.Status.ControllingFaction))
				}
				switch state.Status.ControllingFaction {
				case "0":
					sfxs = append(sfxs, "n-capture")
				case "1":
					sfxs = append(sfxs, "e-capture")
				case "2":
					sfxs = append(sfxs, "r-capture")
				default:
					logW.Warn(fmt.Sprintf("unknown faction '%s'", state.Status.ControllingFaction))
				}
			} else {
				// If the new state was not a change of faction did the number
				// of resoinators change
			}

			if factionChange || forceAmbient {
				ambient := ""
				switch state.Status.ControllingFaction {
				case "0":
					ambient = "n-ambient"
				case "1":
					ambient = "e-ambient"
				case "2":
					ambient = "r-ambient"
				default:
					logW.Warn(fmt.Sprintf("unknown faction '%s'", state.Status.ControllingFaction))
				}
				forceAmbient = false
				select {
				case ambientC <- ambient:
				case <-time.After(3 * time.Second):
				}
			}

			// Check for sound effects that need to be played
			if len(sfxs) != 0 {
				select {
				case sfxC <- sfxs:
				case <-time.After(3 * time.Second):
				}
			}
			// Process the state updates into arduino CMDs and then send these to
			// the arduinos that are listening and our associated with the home portal
			// in any functional capacity
			if homePortal == state.Status.Title {
				cmd := make([]byte, 0, 32)
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
				// Health values are encoded percentages, space for 0%
				resHealth := []byte{' ', ' ', ' ', ' ', ' ', ' ', ' ', ' '}
				// Translate an ascii compass point to a position in the resonators array
				resPositionMap := map[string]int{"N": 2, "NE": 1, "E": 0, "SE": 7, "S": 6, "SW": 5, "W": 4, "NW": 3}

				for _, res := range state.Status.Resonators {
					if position, ok := resPositionMap[res.Position]; ok {
						// After we have the position set the character in the resCmd for that
						// position to the single ASCII digit the represents the level of the
						// resonator
						resCmd[position] = strconv.Itoa(res.Level)[0]
						resHealth[position] = encodePercent(res.Health)
					}
				}

				cmd = append(cmd, resCmd...)
				cmd = append(cmd, encodePercent(state.Status.Health))
				cmd = append(cmd, resHealth...)

				// Mods array handling
				mods := []byte{' ', ' ', ' ', ' '}
				modsMap := map[string]byte{"FA": '0', "HS-C": '1',
					"HS-R": '2', "HS-VR": '3', "LA-R ": '4', "LA-VR": '5',
					"SBUL": '6', "MH-C": '7', "MH-R": '8', "MH-VR": '9',
					"PS-C": 'A', "PS-R": 'B', "PS-VR": 'C', "AXA": 'D',
					"T": 'E'}
				for i, mod := range state.Status.Mods {
					if code, ok := modsMap[mod]; ok {
						mods[i] = code
					}
				}
				cmd = append(cmd, mods...)

				// After printing the overall health output the per resonator health wih delimiters
				cmd = append(cmd, '\n')

				devices := getRunningDevices(homePortal)
				logW.Trace(fmt.Sprintf("sending data to %d devices", len(devices)))

				devicesSent := []string{}

				for _, device := range devices {
					func() {

						defer func() {
							if nil != recover() {
								stopRunningDevice(homePortal, device.devName)
							}
						}()

						if err := device.sendCmd(cmd); err != nil {
							logW.Warn(fmt.Sprintf("%q ➡  device %s role '%s' got an error %s, taking device offline", cmd, device.devName, device.role, err.Error()))
							stopRunningDevice(homePortal, device.devName)
							return
						}
						devicesSent = append(devicesSent, device.devName)
					}()
				}
				logW.Info(fmt.Sprintf("%q ➡ %v", cmd, devicesSent))
			}

			// Save the new state as the last known state
			lastState[state.Status.Title] = state

		case <-quitC:
			return
		}
	}
}
