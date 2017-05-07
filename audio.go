package main

// This module is responsible for driving the audio
// output of this project.
//
// The audio portion of this project requires that
// files be converted to aiff files in 2 channel,
// 44100 Hz, pcm_s16le (16 Bit Signed Little Endian).
//
// The conversion from ogg format files to this format
// can be done using the libav-tools package installed
// using "sudo apt-get install libav-tools".  The
// conversion is done using a command line such as,
// "avconv -i assets/sounds/e-ambient.ogg -ar 44100 -ac 2 -acodec pcm_s16le assets/sounds/e-ambient.aiff".
//
// Playback using the same tools for testing purposes
// can be done using
// "aplay -f S16_LE -c 2 -r 44100 assets/sounds/e-ambient.aiff"

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/karlmutch/termtables"

	"github.com/cvanderschuere/alsa-go"
	"github.com/xlab/vorbis-go/decoder"
)

var (
	audioDir = flag.String("audioDir", "assets/sounds", "The directory in which the audio OGG formatted event files can be found")
)

func initAudio(ambientC <-chan string, sfxC <-chan string, quitC <-chan bool) (err error) {

	printDecoderInfo()

	go runAudio(ambientC, sfxC, quitC)

	return nil
}

// Sounds possible at this point
//
// e-ambient, r-ambient, n-ambient
//
// e-capture, r-capture, n-capture
// e-loss, r-loss, n-loss
// e-loss, r-loss, n-loss
// e-resonator-deployed, r-resonator-deployed, n-resonator-deployed
// e-resonator-destroyed, r-resonator-destroyed, n-resonator-destroyed

func runAudio(ambientC <-chan string, sfxC <-chan string, quitC <-chan bool) {

	go playAmbient(ambientC, quitC)

	for {
		select {

		case fn := <-sfxC:
			logW.Debug("playing %s", fn)
		case <-quitC:
			return
		}
	}
}

type ambientFP struct {
	fp   string
	file *os.File
	sync.Mutex
}

func playAmbient(ambientC <-chan string, quitC <-chan bool) {

	ambient := ambientFP{}

	go func() {
		for {
			select {
			case fn := <-ambientC:
				ambient.Lock()
				ambient.fp = filepath.Join(*audioDir, fn)
				ambient.fp += ".aiff"
				ambient.Unlock()
			case <-quitC:
				return
			}
		}
	}()

	//Open ALSA pipe
	controlChan := make(chan bool)
	defer func() {
		controlChan <- false
		close(controlChan)
	}()
	dataChan := make(chan alsa.AudioData, 100)
	defer close(dataChan)

	//Create stream
	streamChan := alsa.Init(controlChan)
	current_stream := alsa.AudioStream{Channels: 2, Rate: int(44100), SampleFormat: alsa.INT16_TYPE, DataStream: dataChan}

	streamChan <- current_stream

	data := make([]byte, 8192)

	func() {
		fp := ""
		err := errors.New("")
		for {
			ambient.Lock()
			if fp != ambient.fp {
				if ambient.file != nil {
					logW.Debug(fmt.Sprintf("playback of %s stopped", ambient.fp))
					ambient.file.Close()
					ambient.file = nil
				}
				if ambient.fp != "" {
					if ambient.file, err = os.Open(ambient.fp); err != nil {
						logW.Warn(fmt.Sprintf("ambient file %s open failed due to %s, clearing request", ambient.fp, err.Error()))
						ambient.fp = ""
						continue
					}
					fp = ambient.fp
					logW.Debug(fmt.Sprintf("playback of %s starting", ambient.fp))
				}

			}
			ambient.Unlock()
			if ambient.file == nil {
				select {
				case <-time.After(250 * time.Millisecond):
				}
				continue
			}

			data = data[:cap(data)]
			n, err := ambient.file.Read(data)
			if err != nil {
				if err == io.EOF {
					ambient.file.Seek(0, 0)
					logW.Trace(fmt.Sprintf("rewound %s", fp))
					continue
				}
				logW.Warn(err.Error())
				continue
			}
			data = data[:n]

			select {
			case current_stream.DataStream <- data:
			case <-quitC:
				return
			}
		}
	}()
}

func testTone() {
	audioC := make(chan alsa.AudioData, 100)
	controlC := make(chan bool)

	streamC := alsa.Init(controlC)
	defer close(streamC)

	//Send stream
	streamC <- alsa.AudioStream{Channels: 2, Rate: 4410, SampleFormat: alsa.INT16_TYPE, DataStream: audioC}

	//Create sample to play
	b := []byte{0x18, 0x2d, 0x44, 0x54, 0xfb, 0x21, 0x09, 0x40} //PI

	for i := 0; i < 5; i++ {
		b = append(b, b...)
	}

	logW.Trace("start tone")
	stopAt := time.Now().Add(1 * time.Second)

	for {
		audioC <- b
		if time.Now().After(stopAt) {
			break
		}
	}

	logW.Trace("end tone")
}

func printDecoderInfo() {
	table := termtables.CreateTable()
	table.UTF8Box()

	filepath.Walk(*audioDir,
		func(fp string, fi os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if fi.IsDir() {
				return nil // not a file ignore
			}
			matched, err := filepath.Match("*.ogg", fi.Name())
			if err != nil {
				return err
			}
			if matched {
				if err = decoderInfo(fp, table); err != nil {
					logW.Warn(fmt.Sprintf("audio file could not be decoded due to %s", err.Error()), "error", err)
				}
			}
			return nil
		})

	for _, aLine := range strings.Split(table.Render(), "\n") {
		logW.Debug(aLine)
	}
}

func decoderInfo(input string, table *termtables.Table) (err error) {

	stream, err := os.Open(input)
	if err != nil {
		return err
	}
	defer stream.Close()

	dec, err := decoder.New(stream, 2048)
	if err != nil {
		return err
	}
	defer dec.Close()

	fileInfoTable(input, dec.Info(), table)

	return nil
}

func fileInfoTable(name string, info decoder.Info, table *termtables.Table) {
	if !table.IsEmpty() {
		table.AddSeparator()
	}
	var empty []interface{}
	heading := termtables.CreateRow(empty)
	heading.AddCell(
		termtables.CreateCell(name, &termtables.CellStyle{Alignment: termtables.AlignCenter, ColSpan: 2}),
	)
	table.InsertRow(heading)
	table.AddSeparator()

	for _, comment := range info.Comments {
		parts := strings.Split(comment, "=")
		if row := table.AddRow(parts[0]); len(parts) > 1 {
			row.AddCell(parts[1])
		}
	}
	table.AddRow("Bitstream", fmt.Sprintf("%d channel, %.1fHz", info.Channels, info.SampleRate))
	table.AddRow("Encoded by", info.Vendor)
}
