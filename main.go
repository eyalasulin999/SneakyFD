package main

import (
	"time"

	"sneakyfd/config"
	"sneakyfd/internal/handler"
	"sneakyfd/internal/logger"
	"sneakyfd/internal/monitor"
)

func main() {
	logger.Init()
	logger.Log.Info().Msg("#!# SneakyFD Started #!#")
	logger.LogConfig()

	m := monitor.Monitor{DstPorts: config.DstPorts, SrcPorts: config.SrcPorts, CheckInterval: config.CheckInterval}
	err := m.Init()
	if err != nil {
		logger.Log.Error().
			Err(err).
			Msg("Init monitor failed")
		return
	}

	logger.Log.Info().Msg("Monitors for new connections")
	for {
		socks, err := m.GetNewEstablishedSockets()
		if err != nil {
			logger.Log.Error().
				Err(err).
				Msg("Get new connections failed")
		}

		for _, sock := range socks {
			go handler.Handle(sock)
		}
		time.Sleep(config.CheckInterval)
	}
}
