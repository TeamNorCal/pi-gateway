package main

// This file contains the implementation of an arduino interface

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/tarm/serial"
)

var cmd = `#!/bin/bash
for sysdevpath in $(find /sys/bus/usb/devices/usb*/ -name dev); do 
(
syspath="${sysdevpath%/dev}"
devname="$(udevadm info -q name -p $syspath)"
[[ "$devname" == "bus/"* ]] && continue
eval "$(udevadm info -q property --export -p $syspath)"
[[ -z "$ID_SERIAL" ]] && continue
echo -e "/dev/$devname - $ID_SERIAL\n"
)
done `

func sendError(timeout time.Duration, err error, errorC chan<- error) {
	select {
	case <-time.After(timeout):
	case errorC <- err:
	}
}

func run(timeout time.Duration, outputC chan string, errorC chan error, command string, args ...string) {

	defer func() {
		errorC <- nil
	}()

	// instantiate new command
	cmd := exec.Command(command, args...)

	// get pipe to standard output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		sendError(timeout, err, errorC)
		return
	}

	// start process via command
	if err := cmd.Start(); err != nil {
		sendError(timeout, err, errorC)
		return
	}

	// setup a buffer to capture standard output
	var buf bytes.Buffer

	// create a channel to capture any errors from wait
	done := make(chan error)
	go func() {
		if _, err := buf.ReadFrom(stdout); err != nil {
			sendError(timeout, err, errorC)
		}
		done <- cmd.Wait()
	}()

	// block on select, and switch based on actions received
	select {
	case <-time.After(timeout):
		if err := cmd.Process.Kill(); err != nil {
			sendError(timeout, err, errorC)
			return
		}
		sendError(timeout, fmt.Errorf("process timed out"), errorC)
		return
	case err := <-done:
		if err != nil {
			sendError(timeout, err, errorC)
			close(done)

		}
		outputC <- buf.String()
	}
}

// findArduinos locates devices that implement the Arduino
// serial connection.
//
// This function returns a collection of the devices
// that are likely candidates for Arduino.
//
func findArduinos() (devices [][]string, err error) {

	// Create a script to do some device discovery
	tmpfile, err := ioutil.TempFile("", "example")
	if err != nil {
		return nil, err
	}

	defer os.Remove(tmpfile.Name()) // clean up

	if _, err = tmpfile.Write([]byte(cmd)); err != nil {
		return nil, err
	}
	os.Chmod(tmpfile.Name(), 0700)
	if err = tmpfile.Close(); err != nil {
		return nil, err
	}

	devices = [][]string{}

	outputC := make(chan string, 1)
	defer close(outputC)

	errorC := make(chan error, 1)
	defer close(errorC)

	go run(time.Duration(5*time.Second), outputC, errorC, tmpfile.Name())

	for {
		select {
		case dev := <-outputC:
			for _, line := range strings.Split(dev, "\n") {
				if strings.Contains(line, "arduino") {
					details := strings.Split(line, " -")
					serial := strings.Split(line, "_")
					devices = append(devices, []string{details[0], strings.TrimSpace(serial[len(serial)-1])})
					continue
				}
				if strings.Contains(line, "ttyUSB") && strings.Contains(line, "USB_UART") {
					details := strings.Split(line, " -")
					lineParts := strings.Split(line, " -/")
					if len(lineParts) < 2 {
						logW.Warn(fmt.Sprintf("Unexpected device line format %s", line))
						continue
					}
					serial := lineParts[1]
					devices = append(devices, []string{details[0], strings.TrimSpace(serial)})
					continue
				}
			}
		case err = <-errorC:
			if err == nil {
				return devices, nil
			}
			return nil, err
		}
	}
}

type arduino struct {
	port    *serial.Port
	portal  string // The name of ingress portal that this control device is associated with
	devName string // The tty style device name
	role    string // The type of arduino that is present, core, or resonator cluster
}

// startDevice is used to start an individual arduino USB Serial device
//
func startDevice(portalName string, devName string) (device *arduino, err error) {

	device = &arduino{}

	device.port, err = serial.OpenPort(&serial.Config{Name: devName, Baud: 115200, ReadTimeout: time.Duration(time.Second * 5)})

	if err != nil {
		logW.Error(fmt.Sprintf("unable to open arduino at %s due to %s", devName, err.Error()), "error", err)
		return nil, err
	}

	// Let the device stabilize before continuing
	select {
	case <-time.After(2 * time.Second):
	}

	if device.role, err = device.ping(); err != nil {
		device.close()

		logW.Error(fmt.Sprintf("unable to ping arduino at %s due to %s", devName, err.Error()), "error", err)
		return nil, err
	}

	device.devName = devName
	device.portal = portalName
	return device, nil
}

func findDevices() (devices []string) {
	// Parse the comma seperated device list
	devices = strings.Split(*arduinos, ",")

	// If the user did not specify arduinos to be used add then automatically
	if len(devices) == 1 && len(devices[0]) == 0 {
		deviceCatalog, err := findArduinos()
		if err != nil {
			logW.Error(err.Error())
			os.Exit(-3)
		}
		if len(deviceCatalog) == 0 {
			logW.Warn("No arduinos were specified and none could not be found, software will continue running and looking for devices")
		}

		devices = make([]string, 0, len(deviceCatalog))
		for _, attribs := range deviceCatalog {
			logW.Info(fmt.Sprintf("Found arduino '%s' serial # '%s'", attribs[0], attribs[1]), "arduinoDevice", attribs[0], "audrinoSerial", attribs[1])
			devices = append(devices, attribs[0])
		}
	}
	return devices
}

func (dev *arduino) close() (err error) {
	defer func() {
		dev.port = nil
	}()

	return dev.port.Close()
}

func (dev *arduino) ping() (line string, err error) {

	dev.port.Flush()

	n, err := dev.port.Write([]byte("*\n"))
	if err != nil {
		return line, err
	}
	if n != 2 {
		logW.Warn(fmt.Sprintf("%d bytes written out of %d", n, 2))
	}

	buf := make([]byte, 256)
	reader := bufio.NewReader(dev.port)
	buf, err = reader.ReadBytes('\x0a')

	return strings.TrimSpace(string(buf)), nil
}

func (dev *arduino) sendCmd(cmd []byte) (err error) {

	// TODO Add an incremental write loop for serial devices
	n, err := dev.port.Write(cmd)
	if err != nil {
		return err
	}
	if n != len(cmd) {
		logW.Warn(fmt.Sprintf("%d bytes written out of %d", n, len(cmd)))
	}

	return nil
}
