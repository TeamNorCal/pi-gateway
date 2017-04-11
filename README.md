# JSonGateway
A Techthulu JSon to GPIO, I2C and Serial gateway

This repository contains the implementation of a gateway server design to run on the Debian based Raspberry Pi 3.

The requirements for this software are current driven by several data sources from the Magnus Reawakens project hosted by Niatic.

## Requirements

* A photograph and text describing what is believed to be the source of JSon data feed, https://plus.google.com/u/1/+HRichardLoeb/posts/4jzaJ8J9W7c
* The presence of aan RFID capability from the above the purpose and capability of which is unknown, possibly to prevent theft or tampering of the Techthulu device

The gateway is intended to act as a client to a JSon data feed using either stock TCP/IP server ports, or using a Serial port.  In the case of the TCP/IP ports it is assumed that an HTTP 1.1 client will be used.  In the case of the serial device line delimited JSon is assumed to be present.  As time passes details concerning the final host interface for the JSon will become available and this will be modified to meet the changing requirements.

The gateway is also intended to respond to the JSon messages by triggering GPIO I2C pins, or sending serial data to a serial device.

Audio playback will also be supported triggered using JSon from the techthulu module.

## Integration to Audrino software projects

The Team NorCal project has multiple software sub projects that will be implemented by multiple teams.  As a result the interfaces between the Pi and Audrino modules will be done using the simplest integration possible to reduce risk and complexity.

The ASCII output messages for driving the Audrino will be specified by the Audrino project, TBD.

## Interface schematic


ASCII ART HERE


## Builds

Two proposals, python and Go, for an implementation language are afloat and await a decision.

The Golang proposal is to use a compiled language to allow the Pi processor to reduce Pi CPU and memory requirements.  This is raised due to a desire to handle audio, TCP/IP IO, as well as device IO.  In pactice audio demands are taking 1 of the Quad CPUs available.

The JSonGateway project supports cross platform builds for the gateway allowing it to be developed on a non Pi host, including AWS or GCP,  and then binaries targetted at Pi or other ARM processors.  This is done using Docker.

Native builds on the Pi are also supported, this is primarily how the code will be maintained and extended when onsite at Camp Navarro.

## High Level APIs

Various APIs for use with the Pi to support the CPU requirements for a Quad core CPU that can concurrently use the music and audio APIs along with being able to respond to JSon and GPIO requests will be trialed prior to the event.

Audio support - https://github.com/xlab/portaudio-goi, https://github.com/xlab/libvpx-go
Native GPIO - https://github.com/stianeikeland/go-rpio
Serial IO - https://reprage.com/post/using-golang-to-connect-raspberrypi-and-arduino

## Fallback Libraries (Plan B)

https://github.com/kidoman/embd
Custom Hardware (if needed, not likely but an out if we need it) - https://gobot.io/documentation/platforms/raspi/
Googles Low Level Library - https://periph.io/

## Stretch Goals

Support MIDI style music driven by glyphs appearing in the JSon stream, output to standard Pi Audio jack
The USB ports on the pi will be used for line delimited ASCII text messages arriving from the Audrino.

