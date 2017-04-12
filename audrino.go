package main

// This file contains the implementation of an audrino interface

import (
	"io"
	"io/ioutil"
	"strings"

	"github.com/huin/goserial"
)

// findArduinos locates devices that implement the Arduino
// serial connection.
//
// This function returns a collection of the devices
// that are likely candidates for Arduino.
//
func findArduinos() (devices []string) {

	devices = []string{}

	devs, _ := ioutil.ReadDir("/dev")

	for _, f := range devs {
		if strings.Contains(f.Name(), "tty.usbserial") ||
			strings.Contains(f.Name(), "ttyUSB") {
			devices = append(devices, "/dev/"+f.Name())
		}
	}

	return devices
}

type audrino struct {
	port io.ReadWriteCloser
}

// NewAudrino is used to open a connection to an Audrino using
// the serial device passed into the function
//
func NewAudrino(path string) (device *audrino, err error) {

	device = &audrino{}

	device.port, err = goserial.OpenPort(&goserial.Config{Name: path, Baud: 9600})

	if err != nil {
		return nil, err
	}
	return device, nil
}
