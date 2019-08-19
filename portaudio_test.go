package portaudio_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/pipelined/pipe/metric"

	pa "github.com/gordonklaus/portaudio"
	"github.com/pipelined/pipe"
	"github.com/pipelined/portaudio"
	"github.com/pipelined/wav"
	"github.com/stretchr/testify/assert"
)

const (
	bufferSize = 512
	wavSample  = "_testdata/sample.wav"
)

// func TestSink(t *testing.T) {
// 	sink := portaudio.NewSink(nil)

// 	err := sink.Flush("")
// 	assert.Nil(t, err)

// 	fn, err := sink.Sink("", 0, 0)
// 	assert.NotNil(t, fn)
// 	assert.Nil(t, err)

// 	fn, err = sink.Sink("", 44100, 1)
// 	assert.NotNil(t, fn)
// 	assert.Nil(t, err)

// 	err = fn([][]float64{{0, 0, 0}})
// 	assert.Nil(t, err)

// 	err = sink.Flush("")
// 	assert.Nil(t, err)
// }

func TestPADevices(t *testing.T) {
	pa.Initialize()
	devices, err := pa.Devices()
	assert.Nil(t, err)
	fmt.Printf("%+v\n", devices)
	for _, d := range devices {
		fmt.Printf("Device: %v\n", d)
	}

	// TODO: test if device could be used after init/terminate
	pa.Initialize()
	devices, err = pa.Devices()
	assert.Nil(t, err)
	fmt.Printf("%+v\n", devices)
	for _, d := range devices {
		fmt.Printf("Device: %v\n", d)
	}
	pa.Terminate()
	pa.Terminate()
}

func TestPipe(t *testing.T) {
	pa.Initialize()
	devices, err := portaudio.Devices()
	assert.Nil(t, err)
	fmt.Printf("%+v\n", devices)
	for _, d := range devices {
		fmt.Printf("Device: %v\n", d)
	}
	pa.Terminate()
	pa.Initialize()
	pa.Terminate()
	// create pump
	inFile, err := os.Open(wavSample)
	pump := &wav.Pump{ReadSeeker: inFile}
	// create sink with empty device
	sink := portaudio.Sink{}

	l, err := pipe.Line(
		&pipe.Pipe{
			Pump:  pump,
			Sinks: pipe.Sinks(&sink),
		},
	)
	assert.Nil(t, err)

	err = pipe.Wait(l.Run(context.Background(), bufferSize))
	assert.Nil(t, err)
	fmt.Printf("%+v", metric.GetAll())

	// TODO: validate if init/terminate shows new devices
}

func TestDevices(t *testing.T) {
	devices, err := portaudio.Devices()
	assert.Nil(t, err)
	fmt.Printf("%+v\n", devices)
	for _, d := range devices {
		fmt.Printf("Device: %v\n", d)
	}
}
