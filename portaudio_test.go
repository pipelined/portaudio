// +build integration

package portaudio_test

import (
	"context"
	"os"
	"testing"

	"pipelined.dev/audio/portaudio"
	"pipelined.dev/audio/wav"
	"pipelined.dev/pipe"
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
	device, err := portaudio.DefaultOutputDevice()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// create sink with empty device
	line, err := pipe.Routing{
		Source: wav.Source(inFile),
		Sink:   portaudio.Sink(device),
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
