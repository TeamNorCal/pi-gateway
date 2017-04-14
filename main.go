package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mgutz/logxi/v1"
)

var (
	arduinos  = flag.String("arduinos", "", "A list of the preferred arduino devices to be used")
	tecthulhu = flag.String("tecthulhu", "", "Either a serial device, or IP, and optionally port number of the tecthulhu REST server")
	logLevel  = flag.String("loglevel", "debug", "Set the desired log level")
)

// create Logger interface
var logW = log.NewLogger(log.NewConcurrentWriter(os.Stdout), "pi-gateway")

func main() {

	flag.Parse()

	if len(*tecthulhu) == 0 {
		logW.Fatal("No tecthulhu TCP/IP or Serial USB modules were specified")
		os.Exit(-1)
	}

	switch strings.ToLower(*logLevel) {
	case "debug":
		logW.SetLevel(log.LevelDebug)
	case "info":
		logW.SetLevel(log.LevelInfo)
	}

	devices := findDevices()

	startDevices(devices)

	tectC := make(chan interface{}, 1)
	errorC := make(chan error, 1)
	err := startPortal(*tecthulhu, tectC, errorC)

	for {
		select {
		case err = <-errorC:
			logW.Warn(err.Error())
		case state := <-tectC:
			logW.Info(fmt.Sprintf("%v", state))
		}
	}
}
