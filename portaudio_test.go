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
		t.Errorf("PA error: %v", err)
	}

	// create sink with empty device
	p, err := pipe.New(bufferSize,
		pipe.Line{
			Source: wav.Source(inFile),
			Sink:   portaudio.Sink(device),
		},
	)
	if err != nil {
		t.Errorf("pipe error: %v", err)
	}

	err = pipe.Wait(p.Start(context.Background()))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
