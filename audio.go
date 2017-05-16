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
// "aplay -D plug:dmix -f S16_LE -c 2 -r 44100 assets/sounds/e-ambient.aiff"

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cvanderschuere/alsa-go"
)

var (
	audioDir = flag.String("audioDir", "assets/sounds", "The directory in which the audio aiff formatted event files can be found")
)

func initAudio(ambientC <-chan string, sfxC <-chan []string, quitC <-chan bool) (err error) {

	go runAudio(ambientC, sfxC, quitC)

	return nil
}

type effects struct {
	wakeup chan bool
	sfxs   []string
	sync.Mutex
}

var (
	sfxs = effects{
		wakeup: make(chan bool, 1),
		sfxs:   []string{},
	}
)

func playSFX(quitC <-chan bool) {
	//Open ALSA pipe
	controlC := make(chan bool)
	//Create stream
	streamC := alsa.Init(controlC)

	stream := alsa.AudioStream{Channels: 2,
		Rate:         int(44100),
		SampleFormat: alsa.INT16_TYPE,
		DataStream:   make(chan alsa.AudioData, 100),
	}

	streamC <- stream

	defer func() {
		sfxs.Lock()
		close(sfxs.wakeup)
		sfxs.Unlock()
	}()

	sfxs.Lock()
	wakeup := sfxs.wakeup
	sfxs.Unlock()

	for {
		for {
			fp := ""
			sfxs.Lock()
			if len(sfxs.sfxs) != 0 {
				fp = sfxs.sfxs[0]
				sfxs.sfxs = sfxs.sfxs[1:]
			}
			sfxs.Unlock()

			if len(fp) == 0 {
				break
			}
			logW.Debug(fmt.Sprintf("playing %s", fp))

			func() {
				file, err := os.Open(fp)
				if err != nil {
					logW.Warn(fmt.Sprintf("sfx file %s open failed due to %s", fp, err.Error()))
					return
				}
				defer file.Close()

				data := make([]byte, 8192)

				for {
					data = data[:cap(data)]
					n, err := file.Read(data)
					if err != nil {
						if err == io.EOF {
							return
						}
						logW.Warn(err.Error())
						return
					}
					data = data[:n]

					select {
					case stream.DataStream <- append([]byte(nil), data[:n]...):
					case <-quitC:
						return
					}
				}
			}()
		}
		select {
		case <-wakeup:
		case <-time.After(time.Second):
		case <-quitC:
			return
		}
	}
}

// Sounds possible at this point
//
// e-ambient, r-ambient, n-ambient
//
// e-capture, r-capture, n-capture
// e-loss, r-loss, n-loss
// e-resonator-deployed, r-resonator-deployed
// e-resonator-destroyed, r-resonator-destroyed

func runAudio(ambientC <-chan string, sfxC <-chan []string, quitC <-chan bool) {

	go playAmbient(ambientC, quitC)

	go playSFX(quitC)

	for {
		select {

		case fns := <-sfxC:
			if len(fns) != 0 {
				sfxs.Lock()
				for _, fn := range fns {
					sfxs.sfxs = append(sfxs.sfxs, filepath.Join(*audioDir, fn+".aiff"))
				}
				// Wait a maximum of three seconds to wake up the audio
				// player for sound effects
				select {
				case sfxs.wakeup <- true:
				case <-time.After(3 * time.Second):
				}
				sfxs.Unlock()
			}
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
	controlC := make(chan bool)
	//Create stream
	streamC := alsa.Init(controlC)

	stream := alsa.AudioStream{Channels: 2,
		Rate:         int(44100),
		SampleFormat: alsa.INT16_TYPE,
		DataStream:   make(chan alsa.AudioData, 100),
	}

	streamC <- stream

	data := make([]byte, 8192)

	func() {
		fp := ""
		err := errors.New("")
		for {
			ambient.Lock()
			if fp != ambient.fp {
				if ambient.file != nil {
					logW.Debug(fmt.Sprintf("playback of %s stopped", fp))
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

			select {
			case stream.DataStream <- append([]byte(nil), data[:n]...):
			case <-quitC:
				return
			}
		}
	}()
}
