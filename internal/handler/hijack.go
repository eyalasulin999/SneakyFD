package handler

import (
	"context"

	"sneakyfd/internal/hijack"
	"sneakyfd/internal/types"

	"github.com/rs/zerolog"
)

func handleHijack(ctx context.Context, sock types.Socket) (sockProc types.SocketProc, fd int, ok bool) {
	log := zerolog.Ctx(ctx)

	sockProc, err := hijack.LookupSocket(sock.Inode)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Lookup socket process failed")
		return
	}
	log.Info().
		Int("pid", sockProc.PID).
		Int("fd", sockProc.FD).
		Msg("Lookup socket process")

	fd, err = hijack.HijackSocket(sockProc)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Hijack socket failed")
		return
	}
	log.Info().
		Int("fd", fd).
		Msg("Hijacked socket")

	ok = true
	return
}
