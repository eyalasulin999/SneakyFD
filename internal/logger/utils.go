package logger

import (
	"sneakyfd/config"
)

func LogConfig() {
	Log.Info().
    Str("log_level", config.LogLevel.String()).
    Interface("dst_ports", config.DstPorts).
    Interface("src_ports", config.SrcPorts).
    Str("check_interval", config.CheckInterval.String()).
    Interface("markers", config.Markers).
    Bool("kill_process", config.KillProcess).
    Str("wait_timeout", config.WaitProcessTimeout.String()).
    Msg("Configuration")
}