package main

// This module is responsible for driving the audio
// output of this project

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"github.com/karlmutch/termtables"

	"github.com/xlab/portaudio-go/portaudio"
	"github.com/xlab/vorbis-go/decoder"
)

const (
	samplesPerChannel = 2048
	bitDepth          = 16
	sampleFormat      = portaudio.PaFloat32
)

var (
	audioDir = flag.String("audioDir", "assets/sounds", "The directory in which the audio OGG formatted event files can be found")
)

func paError(err portaudio.Error) bool {
	return portaudio.ErrorCode(err) != portaudio.PaNoError
}

func paErrorText(err portaudio.Error) string {
	return portaudio.GetErrorText(err)
}

func initAudio(ambientC <-chan string, sfxC <-chan string, quitC <-chan bool) (err error) {

	printDecoderInfo()

	if paErr := portaudio.Initialize(); paError(paErr) {
		return fmt.Errorf("%#v", paErr)
	}

	go runAudio(ambientC, sfxC, quitC)

	return nil
}

func runAudio(ambientC <-chan string, sfxC <-chan string, quitC <-chan bool) {
	for {
		select {
		case fn := <-ambientC:
			logW.Debug("playing %s on loop", fn)
		case fn := <-sfxC:
			logW.Debug("playing %s", fn)
		case <-quitC:
			if paErr := portaudio.Terminate(); paError(paErr) {
				logW.Warn(fmt.Sprintf("could not stop port audio due to %v", paError))
			}
			return
		}
	}
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

	dec, err := decoder.New(stream, samplesPerChannel)
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

func paCallback(wg *sync.WaitGroup, channels int, samples <-chan [][]float32) portaudio.StreamCallback {
	wg.Add(1)
	return func(_ unsafe.Pointer, output unsafe.Pointer, sampleCount uint,
		_ *portaudio.StreamCallbackTimeInfo, _ portaudio.StreamCallbackFlags, _ unsafe.Pointer) int32 {

		const (
			statusContinue = int32(portaudio.PaContinue)
			statusComplete = int32(portaudio.PaComplete)
		)

		frame, ok := <-samples
		if !ok {
			wg.Done()
			return statusComplete
		}
		if len(frame) > int(sampleCount) {
			frame = frame[:sampleCount]
		}

		idx := 0
		out := (*(*[]float32)(unsafe.Pointer(output)))[:int(sampleCount)*channels]

		for _, sample := range frame {
			if len(sample) > channels {
				sample = sample[:channels]
			}
			for i := range sample {
				out[idx] = sample[i]
				idx++
			}
		}

		return statusContinue
	}
}
