// Package portaudio allows to play audio with Portaudio API.
package portaudio

import (
	"context"
	"fmt"

	"github.com/gordonklaus/portaudio"
	"pipelined.dev/pipe"
	"pipelined.dev/pipe/pooling"
	"pipelined.dev/signal"
)

// Initialize initializes internal portaudio structures. Must be called
// before any other call to this package.
func Initialize() error {
	return portaudio.Initialize()
}

// Terminate cleans up internal portaudio structures. Must be called
// after all streams are closed.
func Terminate() error {
	return portaudio.Terminate()
}

var defaultDevice Device

// DefaultDevice returns system default device.
func DefaultDevice() Device {
	return defaultDevice
}

// Device is the device accessed through portaudio.
type Device struct {
	info *portaudio.DeviceInfo
}

// Sink represets portaudio sink which allows to play audio. If no device
// is provided, the current system default will be used. Sink returns new
// portaudio sink allocator closure.
func Sink(d Device) pipe.SinkAllocatorFunc {
	return func(bufferSize int, props pipe.SignalProperties) (pipe.Sink, error) {
		if d == defaultDevice {
			device, err := defaultOutputDevice()
			if err != nil {
				return pipe.Sink{}, fmt.Errorf("error using default output device: %w", err)
			}
			d = device
		}
		pool := pooling.Get(signal.Allocator{
			Channels: props.Channels,
			Length:   bufferSize,
			Capacity: bufferSize,
		})
		output := make(chan signal.Floating, 1)
		stream, err := portaudio.OpenStream(
			portaudio.StreamParameters{
				Output: portaudio.StreamDeviceParameters{
					Channels: props.Channels,
					Device:   d.info,
					Latency:  d.info.DefaultLowOutputLatency,
				},
				FramesPerBuffer: bufferSize,
				SampleRate:      float64(props.SampleRate),
			},
			сallback(output, pool),
		)
		if err != nil {
			return pipe.Sink{}, fmt.Errorf("error opening PortAudio stream: %w", err)
		}
		if err := stream.Start(); err != nil {
			return pipe.Sink{}, fmt.Errorf("error starting PortAudio stream: %w", err)
		}

		return pipe.Sink{
			SinkFunc:  sink(output, pool),
			FlushFunc: sinkFlusher(stream, output),
		}, nil
	}
}

func сallback(output <-chan signal.Floating, pool *signal.Pool) func([]float32, portaudio.StreamCallbackTimeInfo, portaudio.StreamCallbackFlags) {
	return func(out []float32, timeInfo portaudio.StreamCallbackTimeInfo, flags portaudio.StreamCallbackFlags) {
		select {
		case floats, ok := <-output:
			if !ok {
				return
			}
			signal.ReadFloat32(floats, out)
			pool.PutFloat32(floats)
		default:
		}
	}
}

func sink(output chan<- signal.Floating, pool *signal.Pool) pipe.SinkFunc {
	return func(floats signal.Floating) error {
		buf := pool.GetFloat32()
		signal.FloatingAsFloating(floats, buf)
		output <- buf
		return nil
	}
}

func sinkFlusher(stream *portaudio.Stream, output chan signal.Floating) pipe.FlushFunc {
	return func(context.Context) error {
		close(output)
		if err := stream.Stop(); err != nil {
			return fmt.Errorf("error stopping PortAudio stream: %w", err)
		}
		if err := stream.Close(); err != nil {
			return fmt.Errorf("error closing PortAudio stream: %w", err)
		}
		return nil
	}
}

// Devices return devices available through portaudio. First slice contains
// devices that have input channels, second slice contains devices that
// have output channels and third slice contains devices that doesn't have
// any channels.
func Devices() ([]Device, []Device, []Device, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, nil, nil, fmt.Errorf("error initializing PortAudio: %w", err)
	}
	defer portaudio.Terminate()

	devicesInfo, err := portaudio.Devices()
	if err != nil {
		// error during device refresh, terminate
		if errTerm := portaudio.Terminate(); errTerm != nil {
			// wrap both errors
			return nil, nil, nil, fmt.Errorf("error terminating PortAudio: %w after: %v", errTerm, err)
		}
		// wrap cause error
		return nil, nil, nil, fmt.Errorf("error fetching PortAudio devices: %w", err)
	}
	input := make([]Device, 0)
	output := make([]Device, 0)
	disabled := make([]Device, 0)
	for _, di := range devicesInfo {
		// create device
		d := parseDeviceInfo(di)
		// add device to input
		if di.MaxInputChannels > 0 {
			input = append(input, d)
		}
		// add device to output
		if di.MaxOutputChannels > 0 {
			output = append(output, d)
		}
		// add device to inactive
		if di.MaxInputChannels == 0 && di.MaxOutputChannels == 0 {
			disabled = append(disabled, d)
		}
	}

	return input, output, disabled, nil
}

// defaultOutputDevice returns output device used by system as default at
// the moment.
func defaultOutputDevice() (Device, error) {
	di, err := portaudio.DefaultOutputDevice()
	if err != nil {
		return Device{}, nil
	}
	return parseDeviceInfo(di), nil
}

// defaultInputDevice returns input device used by system as default at the
// moment.
func defaultInputDevice() (Device, error) {
	di, err := portaudio.DefaultInputDevice()
	if err != nil {
		return Device{}, err
	}
	return parseDeviceInfo(di), nil
}

func parseDeviceInfo(di *portaudio.DeviceInfo) Device {
	if di == nil {
		return defaultDevice
	}
	return Device{info: di}
}
