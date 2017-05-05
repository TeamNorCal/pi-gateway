package main

// This module is responsible for driving the audio
// output of this project

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/karlmutch/termtables"

	"github.com/cvanderschuere/alsa-go"
	"github.com/xlab/vorbis-go/decoder"
)

var (
	audioDir = flag.String("audioDir", "assets/sounds", "The directory in which the audio OGG formatted event files can be found")

	controlChan = make(chan bool)
	streamChan  = alsa.Init(controlChan)
)

func initAudio(ambientC <-chan string, sfxC <-chan string, quitC <-chan bool) (err error) {

	printDecoderInfo()

	//Make stream
	dataChan := make(chan alsa.AudioData, 100)
	aStream := alsa.AudioStream{Channels: 2, Rate: 44100, SampleFormat: alsa.INT16_TYPE, DataStream: dataChan}

	//Send stream
	streamChan <- aStream

	go runAudio(ambientC, sfxC, quitC)

	return nil
}

func runAudio(ambientC <-chan string, sfxC <-chan string, quitC <-chan bool) {

	defer close(streamChan)

	for {
		select {
		case fn := <-ambientC:
			logW.Debug("playing %s on loop", fn)
		case fn := <-sfxC:
			logW.Debug("playing %s", fn)
		case <-quitC:
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
