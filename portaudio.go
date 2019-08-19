// Package portaudio allows to play audio with Portaudio API.
package portaudio

import (
	"github.com/gordonklaus/portaudio"
	"github.com/pipelined/signal"
	"golang.org/x/xerrors"
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

	// reset device info with valid device
	var deviceInfo *portaudio.DeviceInfo
	if s.Device == emptyDevice {
		deviceInfo, err = portaudio.DefaultOutputDevice()
	} else {
		deviceInfo, err = refreshDeviceInfo(s.Device)
	}
	// error during device refresh, terminate
	if err != nil {
		if errTerm := portaudio.Terminate(); errTerm != nil {
			// wrap both errors
			return xerrors.Errorf("failed portaudio terminate: %w after: %w", errTerm, err)
		}
		// wrap cause error
		return err
	}

	// update stream params with device
	s.streamParams.Output.Device = deviceInfo
	s.streamParams.Output.Latency = deviceInfo.DefaultLowOutputLatency
	return nil
}

// Flush terminates portaudio structures. It's executed only if Reset didn't return error.
func (s *Sink) Flush(string) (err error) {
	defer func() {
		if errTerm := portaudio.Terminate(); errTerm != nil {
			// wrap termination error
			if err != nil {
				err = xerrors.Errorf("failed portaudio terminate: %w after: %w", errTerm, err)
			} else {
				err = xerrors.Errorf("failed portaudio terminate: %w", err)
			}
		}
	}()
	if s.stream == nil {
		return nil
	}
	err = s.stream.Stop()
	if err != nil {
		return err
	}
	return s.stream.Close()
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
// refreshDeviceInfo MUST be called after successfull portaudio.Initialize.
func refreshDeviceInfo(d Device) (*portaudio.DeviceInfo, error) {
	// retrieve APIs
	apis, err := portaudio.HostApis()
	if err != nil {
		return nil, xerrors.Errorf("failed to retrieve host APIs: %w", err)
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
	return nil, xerrors.Errorf("device %s %s not found", d.api, d.device)
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
