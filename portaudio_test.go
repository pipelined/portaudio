// +build integration

package portaudio_test

import (
	"context"
	"os"
	"testing"

	"pipelined.dev/audio/wav"
	"pipelined.dev/pipe"
	"pipelined.dev/portaudio"
)

const (
	bufferSize = 512
	wavSample  = "_testdata/sample.wav"
)

func TestPipe(t *testing.T) {
	// create pump
	inFile, err := os.Open(wavSample)

	portaudio.Initialize()
	defer portaudio.Terminate()

	// create sink with empty device
	line, err := pipe.Routing{
		Source: wav.Source(inFile),
		Sink:   portaudio.Sink(portaudio.DefaultDevice()),
	}.Line(bufferSize)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	p := pipe.New(context.Background(), pipe.WithLines(line))
	err = p.Wait()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// func TestDevices(t *testing.T) {
// 	devices, err := portaudio.Devices()
// 	assert.Nil(t, err)
// 	assert.NotNil(t, devices)
// }
