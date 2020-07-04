// +build integration

package portaudio_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"pipelined.dev/pipe"
	"pipelined.dev/portaudio"
	"pipelined.dev/wav"
)

const (
	bufferSize = 512
	wavSample  = "_testdata/sample.wav"
)

func TestPipe(t *testing.T) {
	// create pump
	inFile, err := os.Open(wavSample)
	source := wav.Source{ReadSeeker: inFile}
	// create sink with empty device
	sink := portaudio.Sink{}

	line, err := pipe.Routing{
		Source: source.Source(),
		Sink:   sink.Sink(),
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

func TestDevices(t *testing.T) {
	devices, err := portaudio.Devices()
	assert.Nil(t, err)
	assert.NotNil(t, devices)
}
