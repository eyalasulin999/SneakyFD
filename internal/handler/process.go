package handler

import (
	"context"

	"sneakyfd/config"
	"sneakyfd/internal/beacon"
	"sneakyfd/internal/process"

	"github.com/rs/zerolog"
)

func handleProcess(ctx context.Context, pid int) (ok bool) {
	log := zerolog.Ctx(ctx)
	beaconSender := beacon.Ctx(ctx)

	beaconSender.Send(beacon.HANDLE_PROCESS_BEACON)
	log.Info().Msg("Beacon sent - HANDLE PROCESS")

	if config.KillProcess {
		killed := process.KillProcess(pid)
		if !killed {
			log.Error().Msg("Kill process failed, fallback wait process")
			beaconSender.Send(beacon.KILL_PROCESS_FAILED_BEACON)
			log.Info().Msg("Beacon sent - KILL PROCESS FAILED")
		} else {
			log.Info().Msg("Killed process")
			beaconSender.Send(beacon.KILLED_PROCESS_BEACON)
			log.Info().Msg("Beacon sent - KILLED PROCESS")
			ok = true
			return
		}
	}
	log.Info().Msg("Wait process")
	beaconSender.Send(beacon.WAIT_PROCESS_BEACON)
	log.Info().Msg("Beacon sent - WAIT PROCESS")
	w := process.WaitProcess(pid, config.WaitProcessTimeout)
	if w {
		log.Info().Msg("Wait process done")
		beaconSender.Send(beacon.WAIT_PROCESS_DONE_BEACON)
		log.Info().Msg("Beacon sent - WAIT PROCESS DONE")
		ok = true
	} else {
		log.Error().Msg("Wait process timeout")
		beaconSender.Send(beacon.WAIT_PROCESS_TIMEOUT_BEACON)
		log.Info().Msg("Beacon sent - WAIT PROCESS TIMEOUT")
	}

	return
}
