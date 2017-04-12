package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mgutz/logxi/v1"
)

var (
	audrinos  = flag.String("audrinos", "", "A list of the preferred audrino devices to be used")
	tecthulhu = flag.String("tecthulhu", "", "Either a serial device, or IP, and optionally port number of the tecthulhu REST server")
)

func main() {

	flag.Parse()

	if len(*tecthulhu) == 0 {
		log.Fatal("No tecthulhu TCP/IP or Serial USB modules were specified")
		os.Exit(-1)
	}

	// Parse the comma seperated device list
	audrinos := strings.Split(*audrinos, ",")

	// If the user did not specify audrinos to be used add then automatically
	if len(audrinos) == 0 {
		audrinos = findArduinos()
		if len(audrinos) == 0 {
			log.Fatal("No audrinos were specified and could not be found")
			os.Exit(-2)
		}
	}

	tectC := make(chan interface{}, 1)
	errorC := make(chan error, 1)
	err := startPortal(*tecthulhu, tectC, errorC)

	for {
		select {
		case err = <-errorC:
			log.Warn(err.Error())
		case state := <-tectC:
			log.Info(fmt.Sprintf("%v", state))
		}
	}
}
