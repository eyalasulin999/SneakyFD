package handler

import (
	"context"

	"sneakyfd/config"

	"github.com/rs/zerolog"
)

func handleMarker(ctx context.Context, fd int) (ok bool) {
	log := zerolog.Ctx(ctx)

	marked := config.Markers.Check(fd)
	if !marked {
		log.Error().Msg("Socket mark check failed")
	}
	log.Info().Msg("Socket marked")

	ok = true
	return
}
