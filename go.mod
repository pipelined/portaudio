module pipelined.dev/portaudio

require (
	github.com/gordonklaus/portaudio v0.0.0-20180817120803-00e7307ccd93
	github.com/kr/pretty v0.1.0 // indirect
	github.com/stretchr/testify v1.4.0
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	pipelined.dev/pipe v0.8.1
	pipelined.dev/signal v0.7.2
	pipelined.dev/wav v0.4.0
)

go 1.13

replace pipelined.dev/wav => ../wav