// +build integration

package portaudio_test

import (
	"context"
	"os"
	"testing"

	"github.com/pipelined/pipe"
	"github.com/pipelined/portaudio"
	"github.com/pipelined/wav"
	"github.com/stretchr/testify/assert"
)

const (
	bufferSize = 512
	wavSample  = "_testdata/sample.wav"
)

func TestPipe(t *testing.T) {
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
}

func TestDevices(t *testing.T) {
	devices, err := portaudio.Devices()
	assert.Nil(t, err)
	assert.NotNil(t, devices)
}
