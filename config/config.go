package config

import (
	"time"

	"sneakyfd/internal/marker"
	"sneakyfd/internal/types"

	"github.com/rs/zerolog"
)

var LogLevel zerolog.Level = zerolog.InfoLevel
var DstPorts types.Ports = types.Ports{types.FixedPort{Port: 22}}
var SrcPorts types.Ports = types.Ports{types.RangePort{MinPort: 1337, MaxPort: 2337}}
var CheckInterval time.Duration = 1 * time.Second
var Markers marker.Markers = marker.Markers{marker.TCPOptionsMarker{MSS: 1337}}
var KillProcess bool = true
var WaitProcessTimeout time.Duration = 3 * time.Minute
var BeaconMagic []byte = []byte{0xDE, 0xAD, 0xBE, 0xEF}
