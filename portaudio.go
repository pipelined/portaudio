// Package portaudio allows to play audio with Portaudio API.
package portaudio

import (
	"context"
	"fmt"

	"github.com/gordonklaus/portaudio"
	"pipelined.dev/pipe"
	"pipelined.dev/signal"
)

//TODO: ADD API TYPE WITH https://crawshaw.io/blog/sharp-edged-finalizers TO HANDLE CASE WHEN TERMINATE WASN'T EXECUTED

// DefaultOutputDevice returns output device used by system as default at the moment.
func DefaultOutputDevice() (d Device, err error) {
	if err = portaudio.Initialize(); err != nil {
		return
	}
	defer func() {
		if errTerm := portaudio.Terminate(); errTerm != nil {
			err = errTerm
		}
	}()
	var di *portaudio.DeviceInfo
	if di, err = portaudio.DefaultOutputDevice(); err != nil {
		return
	}
	d = parseDeviceInfo(di)
	return
}

// DefaultInputDevice returns input device used by system as default at the moment.
func DefaultInputDevice() (d Device, err error) {
	if err = portaudio.Initialize(); err != nil {
		return
	}
	defer func() {
		if errTerm := portaudio.Terminate(); errTerm != nil {
			err = errTerm
		}
	}()
	var di *portaudio.DeviceInfo
	if di, err = portaudio.DefaultOutputDevice(); err != nil {
		return
	}
	d = parseDeviceInfo(di)
	return
}

var emptyDevice Device

// IO determines the type of device.
type IO int

const (
	// Input is a device that has input channels.
	Input IO = iota
	// Output is a device that has output channels.
	Output
	// Inactive is a device that doesn't have any input or output channels.
	Inactive
)

// Device is the device accessed through portaudio.
type Device struct {
	api         string
	device      string
	outChannels int
	inChannels  int
}

type (
	// Sink represets portaudio sink which allows to play audio.
	// If no device is provided, the current system default will be used.
	Sink struct {
		Device
	}
)

func (s Sink) Sink() pipe.SinkAllocatorFunc {
	return func(bufferSize int, props pipe.SignalProperties) (pipe.Sink, error) {
		if err := portaudio.Initialize(); err != nil {
			return pipe.Sink{}, fmt.Errorf("error initializing PortAudio: %w", err)
		}
		di, err := deviceInfo(s.Device)
		if err != nil {
			// terminate must be called after successful initialize
			if errTerm := portaudio.Terminate(); errTerm != nil {
				// wrap both errors
				return pipe.Sink{}, fmt.Errorf("error terminating PortAudio: %w after: %v", errTerm, err)
			}
			// wrap cause error
			return pipe.Sink{}, fmt.Errorf("error refreshing PortAudio device: %w", err)
		}

		buf := make([]float32, bufferSize*props.Channels)
		streamParams := portaudio.StreamParameters{
			Output: portaudio.StreamDeviceParameters{
				Channels: props.Channels,
				Device:   di,
				Latency:  di.DefaultLowOutputLatency,
			},
			SampleRate: float64(props.SampleRate),
		}
		// open new stream
		stream, err := portaudio.OpenStream(streamParams, &buf)
		if err != nil {
			return pipe.Sink{}, fmt.Errorf("error opening PortAudio stream: %w", err)
		}
		if err := stream.Start(); err != nil {
			return pipe.Sink{}, fmt.Errorf("error starting PortAudio stream: %w", err)
		}

		return pipe.Sink{
			SinkFunc: func(floats signal.Floating) error {
				signal.ReadFloat32(floats, buf)
				if err := stream.Write(); err != nil {
					return fmt.Errorf("error writing PortAudio buffer: %w", err)
				}
				return nil
			},
			FlushFunc: streamFlusher(stream),
		}, nil
	}
}

func streamFlusher(stream *portaudio.Stream) pipe.FlushFunc {
	return func(context.Context) (err error) {
		defer func() {
			if errTerm := portaudio.Terminate(); errTerm != nil {
				// wrap termination error
				if err != nil {
					err = fmt.Errorf("error terminating PortAudio: %w after: %v", errTerm, err)
				} else {
					err = fmt.Errorf("error terminating PortAudio: %w", err)
				}
			}
		}()
		if err = stream.Stop(); err != nil {
			return
		}
		return stream.Close()
	}
}

// Devices return a list of portaudio devices.
func Devices() (map[IO][]Device, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("error initializing PortAudio: %w", err)
	}
	defer portaudio.Terminate()

	devicesInfo, err := portaudio.Devices()
	if err != nil {
		// error during device refresh, terminate
		if errTerm := portaudio.Terminate(); errTerm != nil {
			// wrap both errors
			return nil, fmt.Errorf("error terminating PortAudio: %w after: %v", errTerm, err)
		}
		// wrap cause error
		return nil, fmt.Errorf("error fetching PortAudio devices: %w", err)
	}
	devices := make(map[IO][]Device)
	for _, di := range devicesInfo {
		// create device
		d := parseDeviceInfo(di)
		// add device to input
		if di.MaxInputChannels > 0 {
			devices[Input] = append(devices[Input], d)
		}
		// add device to output
		if di.MaxOutputChannels > 0 {
			devices[Output] = append(devices[Output], d)
		}
		// add device to inactive
		if di.MaxInputChannels == 0 && di.MaxOutputChannels == 0 {
			devices[Inactive] = append(devices[Inactive], d)
		}
	}

	return devices, nil
}

// refresh device info for provided device.
// deviceInfo MUST be called after successfull portaudio.Initialize.
// TODO: rename to fetch device info
func deviceInfo(d Device) (*portaudio.DeviceInfo, error) {
	if d == emptyDevice {
		di, err := portaudio.DefaultOutputDevice()
		if err != nil {
			return nil, fmt.Errorf("error refreshing default PortAudio output device: %w", err)
		}
		return di, nil
	}
	// retrieve APIs
	apis, err := portaudio.HostApis()
	if err != nil {
		return nil, fmt.Errorf("error retrieving PortAudio host APIs: %w", err)
	}

	// find API and device
	var di *portaudio.DeviceInfo
	for _, api := range apis {
		if api.Name == d.api {
			for _, device := range api.Devices {
				if device.Name == d.device {
					di = device
				}
			}
		}
	}
	if di != nil {
		return di, nil
	}
	return nil, fmt.Errorf("device %s %s not found", d.api, d.device)
}

func parseDeviceInfo(di *portaudio.DeviceInfo) Device {
	if di == nil {
		return emptyDevice
	}
	return Device{
		device:      di.Name,
		api:         di.HostApi.Name,
		inChannels:  di.MaxInputChannels,
		outChannels: di.MaxOutputChannels,
	}
}

func (i IO) String() string {
	switch i {
	case Input:
		return "input"
	case Output:
		return "output"
	case Inactive:
		return "inactive"
	}
	return "unknown io type"
}
