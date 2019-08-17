// Package portaudio allows to play audio with Portaudio API.
package portaudio

import (
	"github.com/gordonklaus/portaudio"
	"github.com/pipelined/signal"
	"golang.org/x/xerrors"
)

//TODO: ADD API TYPE WITH https://crawshaw.io/blog/sharp-edged-finalizers TO HANDLE CASE WHEN TERMINATE WASN'T EXECUTED

var (
	// DefaultOutputDevice can be used to utilize default output devices.
	DefaultOutputDevice Device
	// DefaultInputDevice can be used to utilize default input devices.
	DefaultInputDevice Device
)

// IO determines the type of device.
type IO int

const (
	// Input is a device that has input channels.
	Input IO = iota
	// Output is a device that has output channels.
	Output
)

// Device is the device accessed through portaudio.
type Device struct {
	hostAPI     string
	device      string
	outChannels int
	inChannels  int
}

type (
	// Sink represets portaudio sink which allows to play audio using default device.
	Sink struct {
		Device
		streamParams *portaudio.StreamParameters
		stream       *portaudio.Stream
	}
)

// Sink writes the buffer of data to portaudio stream.
// It aslo initilizes a portaudio api with default stream.
func (s *Sink) Sink(sourceID string, sampleRate, numChannels int) (func([][]float64) error, error) {
	var (
		buf               []float32
		currentBufferSize int
	)
	s.streamParams = &portaudio.StreamParameters{
		Output: portaudio.StreamDeviceParameters{
			Channels: numChannels,
		},
		SampleRate: float64(sampleRate),
	}
	return func(b [][]float64) error {
		// buffer size has changed
		if bufferSize := signal.Float64(b).Size(); currentBufferSize != bufferSize {
			currentBufferSize = bufferSize
			buf = make([]float32, bufferSize*numChannels)

			// TODO: HANDLE STREAM CLOSE/OPEN
			// s.stream, err = portaudio.OpenDefaultStream(0, numChannels, float64(sampleRate), bufferSize, &buf)
			stream, err := portaudio.OpenStream(*s.streamParams, &buf)
			if err != nil {
				return err
			}

			err = stream.Start()
			if err != nil {
				return err
			}
			s.stream = stream
		}

		for i := range b[0] {
			for j := range b {
				buf[i*numChannels+j] = float32(b[j][i])
			}
		}
		return s.stream.Write()
	}, nil
}

// Reset sink to use valid portaudio device info.
func (s *Sink) Reset(string) error {
	// reset PA
	err := portaudio.Initialize()
	if err != nil {
		return err
	}

	deviceInfo, err := refreshDeviceInfo(s.Device)
	if err != nil {
		return err
	}
	// reset device info with valid device

	// update stream params with device
	s.streamParams.Output.Device = deviceInfo
	s.streamParams.Output.Latency = deviceInfo.DefaultLowOutputLatency
	return nil
}

// Flush terminates portaudio structures.
func (s *Sink) Flush(string) error {
	if s.stream == nil {
		return nil
	}
	err := s.stream.Stop()
	if err != nil {
		return err
	}
	err = s.stream.Close()
	if err != nil {
		return err
	}

	// TODO: ensure that Terminate is executed (maybe with defer)
	return portaudio.Terminate()
}

// Devices return a list of portaudio devices.
func Devices() (map[IO][]Device, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, err
	}
	defer portaudio.Terminate()

	devicesInfo, err := portaudio.Devices()
	if err != nil {
		return nil, err
	}
	devices := make(map[IO][]Device)
	for _, di := range devicesInfo {
		// create device
		d := Device{
			device:      di.Name,
			hostAPI:     di.HostApi.Name,
			inChannels:  di.MaxInputChannels,
			outChannels: di.MaxOutputChannels,
		}
		// add device to input
		if di.MaxInputChannels > 0 {
			devices[Input] = append(devices[Input], d)
		}
		// add device to output
		if di.MaxOutputChannels > 0 {
			devices[Output] = append(devices[Output], d)
		}
	}

	return devices, nil
}

// refresh device info for provided device.
// refreshDeviceInfo MUST be called after successfull portaudio.Initialize.
func refreshDeviceInfo(d Device) (*portaudio.DeviceInfo, error) {
	switch d {
	case DefaultOutputDevice:
		return portaudio.DefaultOutputDevice()
	case DefaultInputDevice:
		return portaudio.DefaultInputDevice()
	}

	// retrieve APIs
	apis, err := portaudio.HostApis()
	if err != nil {
		return nil, xerrors.Errorf("failed to retrieve host APIs: %w", err)
	}

	// find used API
	var deviceInfo *portaudio.DeviceInfo
	for _, api := range apis {
		if api.Name == d.hostAPI {
			for _, device := range api.Devices {
				if device.Name == d.device {
					deviceInfo = device
				}
			}
		}
	}
	if deviceInfo != nil {
		return deviceInfo, nil
	}
	return nil, xerrors.Errorf("device %s %s not found", d.hostAPI, d.device)
}
