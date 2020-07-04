module pipelined.dev/portaudio

require (
	github.com/gordonklaus/portaudio v0.0.0-20180817120803-00e7307ccd93
	pipelined.dev/audio/wav v0.4.0
	pipelined.dev/pipe v0.8.1
	pipelined.dev/signal v0.7.2
)

go 1.13

replace (
	pipelined.dev/audio/wav => ../wav
	pipelined.dev/pipe => ../pipe
)
