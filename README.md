# pi-gateway
A Techthulu JSon to GPIO, I2C and Serial gateway

This repository contains the implementation of a gateway server design to run on the Debian based Raspberry Pi 3.

The requirements for this software are current driven by several data sources from the Magnus 
Reawakens project hosted by Niatic.

## Requirements

* A photograph and text describing what is believed to be the source of JSon data feed, https://plus.google.com/u/1/+HRichardLoeb/posts/4jzaJ8J9W7c
* The presence of aan RFID capability from the above the purpose and capability of which is unknown, possibly to prevent theft or tampering of the Techthulu device

The gateway is intended to act as a client to a JSon data feed using either stock TCP/IP server 
ports, or using a Serial port.  In the case of the TCP/IP ports it is assumed that an HTTP 1.1 client 
will be used.  In the case of the serial device line delimited JSon is assumed to be present.  As 
time passes details concerning the final host interface for the JSon will become available and 
this will be modified to meet the changing requirements.

The gateway is also intended to respond to the JSon messages by triggering GPIO I2C pins, or 
sending serial data to a serial device.

Audio playback will also be supported triggered using JSon from the techthulu module.

## Integration to Audrino software projects

The Team NorCal project has multiple software sub projects that will be implemented by multiple 
teams.  As a result the interfaces between the Pi and Audrino modules will be done using the 
simplest integration possible to reduce risk and complexity.

The ASCII output messages for driving the Audrino will be specified by the Audrino project.

## Interface schematic


ASCII ART HERE

## ASCII Protocol

The ASCII protocol is used between the pi-gateway and audrinos that successfully respond to
the '*' commands with a Magnus response.  On the console of the pi-gateway when these
devices are detected you will you see a message such as 
"arduino at /dev/ttyACM0 has the role of 'Magnus Resonators Node'".

All messages between the arduinos and the pi-gateway are line delimited.

The basic format of the message is as follows :

Fnnnnnnnn:..d:..r:..r:..r:..r:..r:..r:..r:..r:mmmm:\n

F represents the current faction that holds the portal.  When in uppercase this represents
a change in the owning faction.  'r', or 'R' for resistance, 'n', or 'N' for neutral,
'e', or 'E' for enlightened.

The nnnnnnnn component of our message is an ASCII string of the resonator levels on the
portal arranged started with the eastern point and going counter-clockwise.  Due east 
being in position 0, NW at position 1, north at position 2 and so on.

A delimiter then appears ':' and this is followed by an ASCII formatted number of the
precentage health of the portal.

Then another delimiter, ':', and a set of delimited ASCII formatted numbers each one
representing the health of the resonators starting with N and then going clockwise.

The next set of four ASCII characters contain information about the mods that
are present on the portal.

  - No mod present in this slot
0 - FA Force Amp
1 - HS-C Heat Shield, Common
2 - HS-R Heat Shield, Rare
3 - HS-VR Heat Shield, Very Rare
4 - LA-R Link Amplifier, Rare
5 - LA-VR Link Amplifier, Very Rare
6 - SBUL SoftBank Ultra Link
7 - MH-C MultiHack, Common
8 - MH-R MultiHack, Rare
9 - MH-VR MultiHack, Very Rare
A - PS-C Portal Shield, Common
B - PS-R Portal Shield, Rare
C - PS-VR Portal Sheild, Very Rare
D - AXA AXS Shield
E - T Turret


Finally a delimiter ':' and '\n' character terminates the message.

## Building

Native builds on the Pi are the default , this is primarily how the code will be maintained and extended when 
onsite at Camp Navarro.  When using the Pi 3 builds can be performed by doing a git clone of the guthub 
repo then doing the following

<pre>
cd ~
sudo apt-get install -y wget
wget --quiet -O go.tgz https://storage.googleapis.com/golang/go1.8.1.linux-armv6l.tar.gz
tar xzfo go.tgz
export PATH=$PATH:/home/$USER/go/bin
export GOROOT=/home/$USER/go

cd pi-gateway
export GOPATH=`pwd`
export GOBIN=`pwd`/bin
go get -d . && go build -o bin/pi-gateway .
</pre>

Having done this the binaries will be found in the bin directory of your cloned repo.

The instructions for performing cross platform builds are included inside the Dockerfile.

Two proposals, python and Go, for an implementation language are afloat and await a decision.

The Golang proposal is to use a compiled language to allow the Pi processor to reduce Pi CPU and memory 
requirements.  This is raised due to a desire to handle audio, TCP/IP IO, as well as device IO.  In pactice 
audio demands are taking 1 of the Quad CPUs available.

The pi-gateway project supports cross platform builds for the gateway allowing it to be developed 
on a non Pi host, including AWS or GCP,  and then binaries targetted at Pi or other ARM 
processors.  This is done using Docker.

## Simulators

The number of simulators are available.  All offer pros and cons and also more than one can
be used for testing purposes.

### Self Hosted or Standalone testing

A simulator is also provided for the tecthulhu in the form of JSON files that can be served using a static web server
or a testing web server such as the HttpRoller.

HttpRoller is not application aware and does not need a DSL.  You only need know the tecthulhu payloads
and drop files in to the appropriate directories based upon the time slot you want them served.

HttpRoller provides the ability to change the URLs being served and the content of web pages as a function of time.
This allow complex scenarios such as a portal being attacked and then claimed by a new factor to be simulated
as a loop.

A number of scenarios are included in the pi-gateway repository at pi-gateway/simulator/scenarios/...

<pre>
cd pi-gateway
export GOPATH=`pwd`
export GOBIN=`pwd`/bin
go get github.com/karlmutch/HttpRoller
bin/HttpRoller -listen=127.0.0.1:12345 -path="./simulator/scenarios/default" -window 15s
</pre>

Manually retriving information from the simulator is as simple as:

<pre>
wget -O- --quiet 127.0.0.1:12345/module/status/json
</pre>

This simulator can be run without complex infrastructure and runs with a single binary.

### Public server testing

A number of projects existing on the public internet for serving tecthulhu web pages and can be found at
<pre>http://tecthulhu.boop.blue/...</pre>.  Source code can be found at <pre>https://pypi.python.org/pypi/tecthulhu/1.1</pre>.  
Not all URIs are supported by this implementation and controlling the scenarios for portal states is up to the provider.

A further self hosted simulator can be found at <pre>https://github.com/bbulkow/MagnusFlora</pre>.  This solution 
requires self hosting and is probably the best application aware simulator and comes with an ansible role.  It does however
have lots of moving parts, is complicated to modify DSL based files, deploy, and operate.

## High Level APIs

Various APIs for use with the Pi to support the CPU requirements for a Quad core CPU that can concurrently use the music and audio APIs along with being able to respond to JSon and GPIO requests will be trialed prior to the event.

<pre>
Audio support - https://github.com/xlab/portaudio-goi, https://github.com/xlab/libvpx-go
Native GPIO - https://github.com/stianeikeland/go-rpio
Serial IO - https://reprage.com/post/using-golang-to-connect-raspberrypi-and-arduino
</pre>

## Fallback Libraries (Plan B)

<pre>
https://github.com/kidoman/embd
Custom Hardware (if needed, not likely but an out if we need it)
- https://gobot.io/documentation/platforms/raspi/
Googles Low Level Library - https://periph.io/
</pre>

## Stretch Goals

Support MIDI style music driven by glyphs appearing in the JSon stream, output to standard Pi Audio jack
The USB ports on the pi will be used for line delimited ASCII text messages arriving from the Audrino.

## Additional reference materials

http://investigate.ingress.com/2017/03/16/glyph-music/

#Port Audio needs a decoder to go with it
apt-get install portaudio19-dev
apt-get install libasound-dev
github.com/gordonklaus/portaudio

#Does not work
sudo apt-get install libsndfile-dev
go get github.com/mkb218/gosndfile/sndfile

#libav-tools
apt-get install libav-tools

PulseAudio notes and bluetooth notes at http://blog.mrverrall.co.uk/2013/01/raspberry-pi-a2dp-bluetooth-audio.html
https://gist.github.com/oleq/24e09112b07464acbda1
sudo apt-get install libpulse-dev
https://github.com/mesilliac/pulse-simple/
https://github.com/moriyoshi/pulsego


#Simple but does lots of graphics, has direct OGG example for decoding
sudo apt-get install libsfml-dev
https://bitbucket.org/krepa098/gosfml2/wiki/Home

