package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mgutz/logxi/v1"
)

var (
	arduinos      = flag.String("arduinos", "", "A list of the preferred arduino devices to be used")
	tecthulhus    = flag.String("tecthulhus", "http://127.0.0.1:12345", "A list of either a serial devices usb://dev/ttyAMA10, or http://IP:port numbers/ for the tecthulhu REST/JSon servers to watch")
	concAddress   = flag.String("concentrator", "", "The ZTCP/IP address of a Niantic concetrator if available")
	homeTecthulhu = flag.String("home", "Team NorCal", "The name of the portal which we wish to subscribe to and use to drive our arduinos")
	logLevel      = flag.String("loglevel", "warning", "Set the desired log level")
)

// create Logger interface
var logW = log.NewLogger(log.NewConcurrentWriter(os.Stdout), "pi-gateway")

func main() {

	flag.Parse()

	if len(*tecthulhus) == 0 {
		logW.Fatal("No tecthulhu TCP/IP or Serial USB modules were specified")
		os.Exit(-1)
	}

	// Wait until intialization is over before applying the log level
	logW.SetLevel(log.LevelInfo)

	switch strings.ToLower(*logLevel) {
	case "trace":
		logW.SetLevel(log.LevelTrace)
	case "debug":
		logW.SetLevel(log.LevelDebug)
	case "info":
		logW.SetLevel(log.LevelInfo)
	case "warning", "warn":
		logW.SetLevel(log.LevelWarn)
	case "error", "err":
		logW.SetLevel(log.LevelError)
	case "fatal":
		logW.SetLevel(log.LevelFatal)
	default:
		logW.Error("unrecognized log level specified")
	}

	quitC := make(chan bool, 1)

	// AUdio comes with 2 mixed channels of audio, ambientC is a looped
	// playback that will interrupt ambient playback as a new file name
	// is recieved, and sfxC is a single effect that will interrupt
	// any other playing sfx file
	ambientC := make(chan string, 1)
	sfxC := make(chan []string, 1)

	initAudio(ambientC, sfxC, quitC)

	// portals encapsulate a JSon data feed from ingress techthulu nodes
	//
	tectC := make(chan *portalStatus, 3)
	errorC := make(chan error, 1)

	portals := strings.Split(*tecthulhus, ",")

	if len(*concAddress) != 0 {
		conc := &concentrator{
			url:     *concAddress,
			statusC: tectC,
			errorC:  errorC,
		}
		go conc.startPortals(quitC)
	} else {
		tec := &tecthulhu{
			url:     portals[0],
			statusC: tectC,
			errorC:  errorC,
		}
		go tec.startPortals(quitC)
	}

	// Create a channel over which notifications will be sent for new
	// arduino devices that are detected, the gateway listens
	// for these and uses them for sending updates to the portal state
	//
	go plugAndPlay(quitC)

	// The gateway bridges the status reports from portals down to arduinos
	// using the serial protocols defined by the arduino team
	//
	go startGateway(*homeTecthulhu, tectC, ambientC, sfxC, quitC)

	// If someone presses ctrl C then close our quitc channel to shutdown the system
	// in an orderly way especially when dealing with device handles for the serial IO
	//
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		select {
		case <-quitC:
		case <-sigC:
			close(quitC)
		}
	}()

	// Having started all of the IO interfaces concurrently simply loop
	// waiting for any changes in state while the processing occurs
	// in other threads
	for {
		select {
		case err := <-errorC:
			logW.Warn(err.Error())
		case <-quitC:
			for _, dev := range getRunningDevices(*homeTecthulhu) {
				stopRunningDevice(*homeTecthulhu, dev.devName)
				logW.Warn(fmt.Sprintf("closing portal %s attached to device %s acting as a %s", *homeTecthulhu, dev.devName, dev.role))
			}
			return
		}
	}
}
